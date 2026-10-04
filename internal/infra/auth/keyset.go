package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

const (
	defaultJWKSMinRefreshInterval = 30 * time.Second
	defaultJWKSMaxAge             = 5 * time.Minute
	maxJWKSResponseBytes          = 1 << 20
)

// boundedRemoteKeySet keeps JWKS refreshes bounded in both directions:
// unknown kids cannot force an HTTP request per token, while cached keys have
// a hard maximum age so removed keys do not remain trusted indefinitely.
type boundedRemoteKeySet struct {
	jwksURL      string
	client       *http.Client
	algorithms   []jose.SignatureAlgorithm
	allowedAlgs  map[string]struct{}
	minRefresh   time.Duration
	maxAge       time.Duration
	refreshMu    sync.Mutex
	mu           sync.RWMutex
	keys         []jose.JSONWebKey
	fetchedAt    time.Time
	lastAttempt  time.Time
	lastFetchErr error
}

func newBoundedRemoteKeySet(
	jwksURL string,
	client *http.Client,
	signingAlgorithms []string,
	minRefresh time.Duration,
	maxAge time.Duration,
) *boundedRemoteKeySet {
	algs := make([]jose.SignatureAlgorithm, 0, len(signingAlgorithms))
	allowed := make(map[string]struct{}, len(signingAlgorithms))
	for _, algorithm := range signingAlgorithms {
		algs = append(algs, jose.SignatureAlgorithm(algorithm))
		allowed[algorithm] = struct{}{}
	}
	return &boundedRemoteKeySet{
		jwksURL:     jwksURL,
		client:      client,
		algorithms:  algs,
		allowedAlgs: allowed,
		minRefresh:  minRefresh,
		maxAge:      maxAge,
	}
}

func (r *boundedRemoteKeySet) VerifySignature(ctx context.Context, rawJWT string) ([]byte, error) {
	jws, err := jose.ParseSigned(rawJWT, r.algorithms)
	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %w", err)
	}

	keys, fetchedAt, _, _ := r.snapshot()
	if len(keys) == 0 || r.maxAge <= 0 || time.Since(fetchedAt) >= r.maxAge {
		keys, err = r.refresh(ctx, true)
		if err != nil {
			return nil, fmt.Errorf("refreshing JWKS: %w", err)
		}
	}

	if payload, ok := verifyWithKeys(jws, keys); ok {
		return payload, nil
	}

	// A new kid may indicate a normal key rotation. Refresh at most once per
	// minimum interval so arbitrary kids cannot amplify traffic to the IdP.
	keys, err = r.refresh(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("refreshing JWKS: %w", err)
	}
	if payload, ok := verifyWithKeys(jws, keys); ok {
		return payload, nil
	}
	return nil, errors.New("failed to verify token signature")
}

func verifyWithKeys(jws *jose.JSONWebSignature, keys []jose.JSONWebKey) ([]byte, bool) {
	keyID := ""
	if len(jws.Signatures) > 0 {
		keyID = jws.Signatures[0].Header.KeyID
	}
	for _, key := range keys {
		if keyID != "" && key.KeyID != keyID {
			continue
		}
		if payload, err := jws.Verify(&key); err == nil {
			return payload, true
		}
	}
	return nil, false
}

func (r *boundedRemoteKeySet) snapshot() ([]jose.JSONWebKey, time.Time, time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]jose.JSONWebKey(nil), r.keys...), r.fetchedAt, r.lastAttempt, r.lastFetchErr
}

func (r *boundedRemoteKeySet) refresh(ctx context.Context, requireFresh bool) ([]jose.JSONWebKey, error) {
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()

	keys, fetchedAt, lastAttempt, lastErr := r.snapshot()
	now := time.Now()
	fresh := len(keys) > 0 && r.maxAge > 0 && now.Sub(fetchedAt) < r.maxAge
	if requireFresh && fresh {
		return keys, nil
	}

	if !lastAttempt.IsZero() && now.Sub(lastAttempt) < r.minRefresh {
		if requireFresh && !fresh {
			if lastErr != nil {
				return keys, lastErr
			}
			return keys, errors.New("JWKS refresh is temporarily throttled")
		}
		return keys, nil
	}

	r.mu.Lock()
	r.lastAttempt = now
	r.mu.Unlock()

	newKeys, err := r.fetchKeys(ctx)

	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.lastFetchErr = err
		return append([]jose.JSONWebKey(nil), r.keys...), err
	}

	r.keys = newKeys
	r.fetchedAt = now
	r.lastFetchErr = nil
	return append([]jose.JSONWebKey(nil), r.keys...), nil
}

func (r *boundedRemoteKeySet) fetchKeys(ctx context.Context) ([]jose.JSONWebKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating JWKS request: %w", err)
	}
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading JWKS response: %w", err)
	}
	if len(body) > maxJWKSResponseBytes {
		return nil, fmt.Errorf("JWKS response exceeds %d bytes", maxJWKSResponseBytes)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned %s", resp.Status)
	}

	var raw struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding JWKS: %w", err)
	}

	keys := make([]jose.JSONWebKey, 0, len(raw.Keys))
	for _, rawKey := range raw.Keys {
		var metadata struct {
			Algorithm string `json:"alg"`
			Use       string `json:"use"`
		}
		if err := json.Unmarshal(rawKey, &metadata); err != nil {
			return nil, fmt.Errorf("decoding JWK metadata: %w", err)
		}
		if metadata.Use != "" && metadata.Use != "sig" {
			continue
		}
		if metadata.Algorithm != "" {
			if _, ok := r.allowedAlgs[metadata.Algorithm]; !ok {
				continue
			}
		}

		var key jose.JSONWebKey
		if err := json.Unmarshal(rawKey, &key); err != nil {
			if errors.Is(err, jose.ErrUnsupportedKeyType) {
				continue
			}
			return nil, fmt.Errorf("decoding JWK: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, nil
}
