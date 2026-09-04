package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	infraauth "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/auth"
)

const (
	testIssuer   = "https://identity.example.com"
	testAudience = "https://mcp.example.com/sysdig-mcp-server"
)

type accessTokenClaims struct {
	Scope string `json:"scope,omitempty"`
	SCP   any    `json:"scp,omitempty"`
}

func TestJWTVerifier(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	keyID := "test-key"
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       &privateKey.PublicKey,
		KeyID:     keyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}}}
	jwksHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Errorf("encoding JWKS: %v", err)
		}
	})
	jwksServer := httptest.NewServer(jwksHandler)
	defer jwksServer.Close()

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID),
	)
	if err != nil {
		t.Fatalf("creating signer: %v", err)
	}

	newToken := func(issuer string, audience jwt.Audience, expiry time.Time, claims accessTokenClaims) string {
		t.Helper()
		rawToken, err := jwt.Signed(signer).
			Claims(jwt.Claims{
				Issuer:   issuer,
				Audience: audience,
				Expiry:   jwt.NewNumericDate(expiry),
			}).
			Claims(claims).
			Serialize()
		if err != nil {
			t.Fatalf("signing token: %v", err)
		}
		return rawToken
	}

	tests := []struct {
		name              string
		issuer            string
		audience          jwt.Audience
		expiry            time.Time
		claims            accessTokenClaims
		signingAlgorithms []string
		requiredScopes    []string
		wantError         bool
		wantScopeError    bool
	}{
		{
			name:              "valid scope claim",
			issuer:            testIssuer,
			audience:          jwt.Audience{testAudience},
			expiry:            time.Now().Add(time.Hour),
			claims:            accessTokenClaims{Scope: "openid mcp:tools"},
			signingAlgorithms: []string{"RS256"},
			requiredScopes:    []string{"mcp:tools"},
		},
		{
			name:              "valid scp claim",
			issuer:            testIssuer,
			audience:          jwt.Audience{"another-audience", testAudience},
			expiry:            time.Now().Add(time.Hour),
			claims:            accessTokenClaims{SCP: []string{"mcp:tools", "profile"}},
			signingAlgorithms: []string{"RS256"},
			requiredScopes:    []string{"mcp:tools", "profile"},
		},
		{
			name:              "valid space-delimited scp claim",
			issuer:            testIssuer,
			audience:          jwt.Audience{testAudience},
			expiry:            time.Now().Add(time.Hour),
			claims:            accessTokenClaims{SCP: "mcp:tools profile"},
			signingAlgorithms: []string{"RS256"},
			requiredScopes:    []string{"mcp:tools", "profile"},
		},
		{
			name:              "wrong issuer",
			issuer:            "https://attacker.example.com",
			audience:          jwt.Audience{testAudience},
			expiry:            time.Now().Add(time.Hour),
			signingAlgorithms: []string{"RS256"},
			wantError:         true,
		},
		{
			name:              "wrong audience",
			issuer:            testIssuer,
			audience:          jwt.Audience{"another-audience"},
			expiry:            time.Now().Add(time.Hour),
			signingAlgorithms: []string{"RS256"},
			wantError:         true,
		},
		{
			name:              "expired",
			issuer:            testIssuer,
			audience:          jwt.Audience{testAudience},
			expiry:            time.Now().Add(-time.Hour),
			signingAlgorithms: []string{"RS256"},
			wantError:         true,
		},
		{
			name:              "missing required scope",
			issuer:            testIssuer,
			audience:          jwt.Audience{testAudience},
			expiry:            time.Now().Add(time.Hour),
			claims:            accessTokenClaims{Scope: "openid"},
			signingAlgorithms: []string{"RS256"},
			requiredScopes:    []string{"mcp:tools"},
			wantError:         true,
			wantScopeError:    true,
		},
		{
			name:              "disallowed signing algorithm",
			issuer:            testIssuer,
			audience:          jwt.Audience{testAudience},
			expiry:            time.Now().Add(time.Hour),
			signingAlgorithms: []string{"ES256"},
			wantError:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rawToken := newToken(test.issuer, test.audience, test.expiry, test.claims)
			verifier := infraauth.NewJWTVerifier(
				context.Background(),
				testIssuer,
				testAudience,
				jwksServer.URL+"/jwks",
				test.signingAlgorithms,
				test.requiredScopes,
			)

			err := verifier.Verify(context.Background(), rawToken)
			if test.wantError && err == nil {
				t.Fatal("expected token verification to fail")
			}
			if test.wantScopeError && !errors.Is(err, infraauth.ErrInsufficientScope) {
				t.Fatalf("expected insufficient scope error, got: %v", err)
			}
			if !test.wantError && err != nil {
				t.Fatalf("expected token verification to succeed: %v", err)
			}
		})
	}

	t.Run("uses a supplied HTTP client for JWKS TLS policy", func(t *testing.T) {
		tlsServer := httptest.NewTLSServer(jwksHandler)
		defer tlsServer.Close()

		rawToken := newToken(
			testIssuer,
			jwt.Audience{testAudience},
			time.Now().Add(time.Hour),
			accessTokenClaims{},
		)

		defaultVerifier := infraauth.NewJWTVerifier(
			context.Background(),
			testIssuer,
			testAudience,
			tlsServer.URL+"/jwks",
			[]string{"RS256"},
			nil,
		)
		if err := defaultVerifier.Verify(context.Background(), rawToken); err == nil {
			t.Fatal("expected the default JWKS client to reject the self-signed certificate")
		}

		customVerifier := infraauth.NewJWTVerifierWithHTTPClient(
			context.Background(),
			testIssuer,
			testAudience,
			tlsServer.URL+"/jwks",
			[]string{"RS256"},
			nil,
			tlsServer.Client(),
		)
		if err := customVerifier.Verify(context.Background(), rawToken); err != nil {
			t.Fatalf("expected custom JWKS HTTP client to be used: %v", err)
		}
	})
}
