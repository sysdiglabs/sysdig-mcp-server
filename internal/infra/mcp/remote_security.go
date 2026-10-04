package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	mcpprotocol "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	infraauth "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/auth"
)

type principalContextKey struct{}

type RemoteSecurity struct {
	verifier           infraauth.TokenVerifier
	allowedOrigins     map[string]struct{}
	corsAllowedOrigins []string
	metadata           server.ProtectedResourceMetadataConfig
	metadataURL        string
}

func NewRemoteSecurity(
	verifier infraauth.TokenVerifier,
	resourceURL string,
	authorizationServer string,
	requiredScopes []string,
	allowedOrigins []string,
) RemoteSecurity {
	origins := make(map[string]struct{}, len(allowedOrigins))
	corsOrigins := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		normalized := normalizeOrigin(origin)
		if _, exists := origins[normalized]; exists {
			continue
		}
		origins[normalized] = struct{}{}
		corsOrigins = append(corsOrigins, normalized)
	}

	metadata := server.ProtectedResourceMetadataConfig{
		Resource:               resourceURL,
		AuthorizationServers:   []string{authorizationServer},
		ScopesSupported:        requiredScopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Sysdig MCP Server",
	}

	return RemoteSecurity{
		verifier:           verifier,
		allowedOrigins:     origins,
		corsAllowedOrigins: corsOrigins,
		metadata:           metadata,
		metadataURL:        protectedResourceMetadataURL(resourceURL),
	}
}

func (s RemoteSecurity) protect(next http.Handler, bindSessions bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin, hasOrigin, ok := requestOrigin(r.Header.Values("Origin"))
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if hasOrigin {
			if _, allowed := s.allowedOrigins[origin]; !allowed {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			if isCORSPreflight(r) {
				next.ServeHTTP(w, r)
				return
			}
			// Authentication runs outside mcp-go's CORS middleware. Add the
			// simple-response headers here too so a browser can read the OAuth
			// challenge from a 401/403 response.
			applyAuthCORSHeaders(w, origin)
		}

		rawToken, ok := bearerToken(r.Header.Values("Authorization"))
		if !ok {
			s.writeUnauthorized(w, "")
			return
		}

		principal, err := s.verifier.Verify(r.Context(), rawToken)
		if err != nil {
			if errors.Is(err, infraauth.ErrInsufficientScope) {
				s.writeInsufficientScope(w)
				return
			}
			s.writeUnauthorized(w, "invalid_token")
			return
		}

		if bindSessions {
			if err := validateRequestSessionOwner(r, principal); err != nil {
				// mcp-go uses 404 for an unknown/invalid session. Preserve that
				// behavior instead of exposing whether a leaked session exists.
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
		}

		ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s RemoteSecurity) corsOrigins() []string {
	return append([]string(nil), s.corsAllowedOrigins...)
}

func (s RemoteSecurity) writeInsufficientScope(w http.ResponseWriter) {
	challenge := fmt.Sprintf(
		`Bearer resource_metadata=%q, error="insufficient_scope", scope=%q`,
		s.metadataURL,
		strings.Join(s.metadata.ScopesSupported, " "),
	)
	w.Header().Set("WWW-Authenticate", challenge)
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

func (s RemoteSecurity) mountMetadata(mux *http.ServeMux) {
	mux.Handle(server.ProtectedResourceMetadataPath(s.metadata.Resource), server.NewProtectedResourceMetadataHandler(s.metadata))
}

func (s RemoteSecurity) writeUnauthorized(w http.ResponseWriter, authError string) {
	challenge := fmt.Sprintf(`Bearer resource_metadata=%q`, s.metadataURL)
	if authError != "" {
		challenge += fmt.Sprintf(`, error=%q`, authError)
	}
	w.Header().Set("WWW-Authenticate", challenge)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func applyAuthCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	appendVary(w.Header(), "Origin")
	w.Header().Set("Access-Control-Expose-Headers", server.HeaderKeySessionID+", WWW-Authenticate")
}

func appendVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, item := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func requestOrigin(values []string) (origin string, present bool, ok bool) {
	if len(values) == 0 {
		return "", false, true
	}
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", true, false
	}
	return normalizeOrigin(values[0]), true, true
}

func normalizeOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return origin
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return u.String()
}

func isCORSPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

func bearerToken(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}

	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}

	return parts[1], true
}

func protectedResourceMetadataURL(resource string) string {
	u, err := url.Parse(resource)
	if err != nil {
		return ""
	}
	u.Path = server.ProtectedResourceMetadataPath(resource)
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func principalFromContext(ctx context.Context) (infraauth.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(infraauth.Principal)
	return principal, ok && principal.Issuer != "" && principal.Subject != ""
}

func validateRequestSessionOwner(r *http.Request, principal infraauth.Principal) error {
	// Protocol 2026-07-28 removed protocol-level sessions. Match mcp-go and
	// ignore a stale Mcp-Session-Id rather than applying legacy ownership
	// semantics to a modern request.
	if r.Header.Get(mcpprotocol.HeaderProtocolVersion) != mcpprotocol.ProtocolVersion20260728 {
		if sessionID := r.Header.Get(mcpprotocol.HeaderSessionID); sessionID != "" {
			if err := validatePrincipalSessionID(sessionID, principal); err != nil {
				return err
			}
		}
	}
	if sessionID := r.URL.Query().Get("sessionId"); sessionID != "" {
		if err := validatePrincipalSessionID(sessionID, principal); err != nil {
			return err
		}
	}
	return nil
}
