package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	infraauth "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/auth"
)

type RemoteSecurity struct {
	verifier       infraauth.TokenVerifier
	allowedOrigins map[string]struct{}
	metadata       server.ProtectedResourceMetadataConfig
	metadataURL    string
}

func NewRemoteSecurity(
	verifier infraauth.TokenVerifier,
	resourceURL string,
	authorizationServer string,
	requiredScopes []string,
	allowedOrigins []string,
) RemoteSecurity {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = struct{}{}
	}

	metadata := server.ProtectedResourceMetadataConfig{
		Resource:               resourceURL,
		AuthorizationServers:   []string{authorizationServer},
		ScopesSupported:        requiredScopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Sysdig MCP Server",
	}

	return RemoteSecurity{
		verifier:       verifier,
		allowedOrigins: origins,
		metadata:       metadata,
		metadataURL:    protectedResourceMetadataURL(resourceURL),
	}
}

func (s RemoteSecurity) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		if origin := r.Header.Get("Origin"); origin != "" {
			if _, allowed := s.allowedOrigins[origin]; !allowed {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id, WWW-Authenticate")

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Last-Event-ID, Mcp-Protocol-Version, Mcp-Session-Id")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		rawToken, ok := bearerToken(r.Header.Values("Authorization"))
		if !ok {
			s.writeUnauthorized(w, "")
			return
		}

		if err := s.verifier.Verify(r.Context(), rawToken); err != nil {
			if errors.Is(err, infraauth.ErrInsufficientScope) {
				s.writeInsufficientScope(w)
				return
			}
			s.writeUnauthorized(w, "invalid_token")
			return
		}

		next.ServeHTTP(w, r)
	})
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
