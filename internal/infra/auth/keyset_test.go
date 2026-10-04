package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

func TestBoundedRemoteKeySetThrottlesUnknownKids(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &privateKey.PublicKey, KeyID: "known", Algorithm: string(jose.RS256), Use: "sig",
	}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	sign := func(kid string) string {
		t.Helper()
		signer, err := jose.NewSigner(
			jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
			(&jose.SignerOptions{}).WithHeader("kid", kid),
		)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := signer.Sign([]byte(`{"sub":"test"}`))
		if err != nil {
			t.Fatal(err)
		}
		serialized, err := raw.CompactSerialize()
		if err != nil {
			t.Fatal(err)
		}
		return serialized
	}

	keySet := newBoundedRemoteKeySet(server.URL, server.Client(), []string{"RS256"}, time.Hour, time.Hour)
	if _, err := keySet.VerifySignature(context.Background(), sign("known")); err != nil {
		t.Fatalf("initial verification failed: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := keySet.VerifySignature(context.Background(), sign("unknown")); err == nil {
			t.Fatal("expected unknown kid to fail")
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("unknown kids triggered %d JWKS fetches, want 1", got)
	}
}

func TestBoundedRemoteKeySetExpiresRemovedKeys(t *testing.T) {
	key1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	key2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.RWMutex
	current := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &key1.PublicKey, KeyID: "key-1", Algorithm: string(jose.RS256), Use: "sig",
	}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		_ = json.NewEncoder(w).Encode(current)
	}))
	defer server.Close()

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key1},
		(&jose.SignerOptions{}).WithHeader("kid", "key-1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign([]byte(`{"sub":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}

	keySet := newBoundedRemoteKeySet(server.URL, server.Client(), []string{"RS256"}, 0, time.Nanosecond)
	if _, err := keySet.VerifySignature(context.Background(), raw); err != nil {
		t.Fatalf("initial verification failed: %v", err)
	}

	mu.Lock()
	current = jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &key2.PublicKey, KeyID: "key-2", Algorithm: string(jose.RS256), Use: "sig",
	}}}
	mu.Unlock()
	time.Sleep(time.Millisecond)

	if _, err := keySet.VerifySignature(context.Background(), raw); err == nil {
		t.Fatal("expected token signed by removed key to fail after cache expiry")
	}
}
