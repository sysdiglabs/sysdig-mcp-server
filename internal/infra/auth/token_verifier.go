package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

const jwksRequestTimeout = 10 * time.Second

// ErrInsufficientScope distinguishes an authenticated token from one that
// lacks the authorization required by this resource.
var ErrInsufficientScope = errors.New("access token has insufficient scope")

// Principal is the verified identity that owns an MCP request/session.
type Principal struct {
	Issuer  string
	Subject string
}

// TokenVerifier validates an access token presented to the MCP server and
// returns the identity that owns any stateful MCP session created by the
// request. Implementations must never forward the token upstream.
type TokenVerifier interface {
	Verify(context.Context, string) (Principal, error)
}

// JWTVerifier validates signed JWT access tokens against a bounded remote JWKS.
// Issuer, audience, expiry, and signing algorithm checks are delegated to the
// OIDC verifier. Optional scopes are checked after cryptographic validation.
type JWTVerifier struct {
	verifier       *oidc.IDTokenVerifier
	requiredScopes []string
}

func NewJWTVerifier(
	ctx context.Context,
	issuer string,
	audience string,
	jwksURL string,
	signingAlgorithms []string,
	requiredScopes []string,
) *JWTVerifier {
	return NewJWTVerifierWithHTTPClient(
		ctx,
		issuer,
		audience,
		jwksURL,
		signingAlgorithms,
		requiredScopes,
		nil,
	)
}

func NewJWTVerifierWithHTTPClient(
	_ context.Context,
	issuer string,
	audience string,
	jwksURL string,
	signingAlgorithms []string,
	requiredScopes []string,
	httpClient *http.Client,
) *JWTVerifier {
	if httpClient == nil {
		httpClient = &http.Client{}
	} else {
		clone := *httpClient
		httpClient = &clone
	}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = jwksRequestTimeout
	}

	keySet := newBoundedRemoteKeySet(
		jwksURL,
		httpClient,
		signingAlgorithms,
		defaultJWKSMinRefreshInterval,
		defaultJWKSMaxAge,
	)
	verifier := oidc.NewVerifier(issuer, keySet, &oidc.Config{
		ClientID:             audience,
		SupportedSigningAlgs: signingAlgorithms,
	})

	return &JWTVerifier{
		verifier:       verifier,
		requiredScopes: slices.Clone(requiredScopes),
	}
}

func (v *JWTVerifier) Verify(ctx context.Context, rawToken string) (Principal, error) {
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return Principal{}, fmt.Errorf("validating access token: %w", err)
	}
	if token.Subject == "" {
		return Principal{}, errors.New("validating access token: missing sub claim")
	}

	principal := Principal{Issuer: token.Issuer, Subject: token.Subject}
	if len(v.requiredScopes) == 0 {
		return principal, nil
	}

	var claims struct {
		Scope json.RawMessage `json:"scope"`
		SCP   json.RawMessage `json:"scp"`
	}
	if err := token.Claims(&claims); err != nil {
		return Principal{}, fmt.Errorf("decoding access token claims: %w", err)
	}

	grantedScopes, err := parseScopeClaim("scope", claims.Scope)
	if err != nil {
		return Principal{}, err
	}
	scpScopes, err := parseScopeClaim("scp", claims.SCP)
	if err != nil {
		return Principal{}, err
	}
	grantedScopes = append(grantedScopes, scpScopes...)

	for _, requiredScope := range v.requiredScopes {
		if !slices.Contains(grantedScopes, requiredScope) {
			return Principal{}, fmt.Errorf("%w: missing %q", ErrInsufficientScope, requiredScope)
		}
	}

	return principal, nil
}

func parseScopeClaim(name string, raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var scopeString string
	if err := json.Unmarshal(raw, &scopeString); err == nil {
		return strings.Fields(scopeString), nil
	}

	var scopeList []string
	if err := json.Unmarshal(raw, &scopeList); err == nil {
		return scopeList, nil
	}

	return nil, fmt.Errorf("decoding %s claim: expected string or string array", name)
}
