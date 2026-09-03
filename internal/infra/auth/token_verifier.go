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

// TokenVerifier validates an access token presented to the MCP server.
// Implementations must never forward the token to an upstream service.
type TokenVerifier interface {
	Verify(context.Context, string) error
}

// JWTVerifier validates signed JWT access tokens against a remote JWKS.
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
	ctx = oidc.ClientContext(ctx, &http.Client{Timeout: jwksRequestTimeout})
	keySet := oidc.NewRemoteKeySet(ctx, jwksURL)
	verifier := oidc.NewVerifier(issuer, keySet, &oidc.Config{
		ClientID:             audience,
		SupportedSigningAlgs: signingAlgorithms,
	})

	return &JWTVerifier{
		verifier:       verifier,
		requiredScopes: slices.Clone(requiredScopes),
	}
}

func (v *JWTVerifier) Verify(ctx context.Context, rawToken string) error {
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return fmt.Errorf("validating access token: %w", err)
	}

	if len(v.requiredScopes) == 0 {
		return nil
	}

	var claims struct {
		Scope string          `json:"scope"`
		SCP   json.RawMessage `json:"scp"`
	}
	if err := token.Claims(&claims); err != nil {
		return fmt.Errorf("decoding access token claims: %w", err)
	}

	grantedScopes := strings.Fields(claims.Scope)
	if len(claims.SCP) > 0 {
		var scopeString string
		if err := json.Unmarshal(claims.SCP, &scopeString); err == nil {
			grantedScopes = append(grantedScopes, strings.Fields(scopeString)...)
		} else {
			var scopeList []string
			if err := json.Unmarshal(claims.SCP, &scopeList); err != nil {
				return fmt.Errorf("decoding scp claim: %w", err)
			}
			grantedScopes = append(grantedScopes, scopeList...)
		}
	}
	for _, requiredScope := range v.requiredScopes {
		if !slices.Contains(grantedScopes, requiredScope) {
			return fmt.Errorf("%w: missing %q", ErrInsufficientScope, requiredScope)
		}
	}

	return nil
}
