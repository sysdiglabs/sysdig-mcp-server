package mcp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	infraauth "github.com/sysdiglabs/sysdig-mcp-server/internal/infra/auth"
)

const principalSessionPrefix = "mcp-session-"

type principalSessionIDManager struct {
	principal infraauth.Principal
}

type principalSessionIDManagerResolver struct{}

func (principalSessionIDManagerResolver) ResolveSessionIdManager(r *http.Request) server.SessionIdManager {
	if r == nil {
		return &principalSessionIDManager{}
	}
	principal, _ := principalFromContext(r.Context())
	return &principalSessionIDManager{principal: principal}
}

func (m *principalSessionIDManager) Generate() string {
	sessionID, err := generatePrincipalSessionID(m.principal)
	if err != nil {
		// SessionIdManager.Generate cannot return an error. A CSPRNG failure is
		// not recoverable without weakening the session boundary, so fail closed.
		panic(fmt.Sprintf("generating MCP session ID: %v", err))
	}
	return sessionID
}

func (m *principalSessionIDManager) Validate(sessionID string) (bool, error) {
	if m.principal.Issuer == "" || m.principal.Subject == "" {
		_, err := parsePrincipalSessionID(sessionID)
		return false, err
	}
	return false, validatePrincipalSessionID(sessionID, m.principal)
}

func (m *principalSessionIDManager) Terminate(sessionID string) (bool, error) {
	return m.Validate(sessionID)
}

func generatePrincipalSessionID(principal infraauth.Principal) (string, error) {
	if principal.Issuer == "" || principal.Subject == "" {
		return "", errors.New("authenticated principal is unavailable")
	}
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(random)
	return principalSessionPrefix + nonce + "." + principalFingerprint(principal), nil
}

func validatePrincipalSessionID(sessionID string, principal infraauth.Principal) error {
	if principal.Issuer == "" || principal.Subject == "" {
		return errors.New("authenticated principal is unavailable")
	}
	fingerprint, err := parsePrincipalSessionID(sessionID)
	if err != nil {
		return err
	}

	expected := principalFingerprint(principal)
	if len(fingerprint) != len(expected) ||
		subtle.ConstantTimeCompare([]byte(fingerprint), []byte(expected)) != 1 {
		return errors.New("session ID belongs to a different principal")
	}
	return nil
}

func parsePrincipalSessionID(sessionID string) (string, error) {
	if !strings.HasPrefix(sessionID, principalSessionPrefix) {
		return "", errors.New("invalid session ID")
	}
	encoded, fingerprint, ok := strings.Cut(strings.TrimPrefix(sessionID, principalSessionPrefix), ".")
	if !ok || encoded == "" || fingerprint == "" || strings.Contains(fingerprint, ".") {
		return "", errors.New("invalid session ID")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(nonce) != 18 {
		return "", errors.New("invalid session ID")
	}
	return fingerprint, nil
}

func principalFingerprint(principal infraauth.Principal) string {
	sum := sha256.Sum256([]byte(principal.Issuer + "\x00" + principal.Subject))
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}
