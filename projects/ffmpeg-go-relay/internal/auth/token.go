package auth

import (
	"fmt"
	"strings"
	"sync"
)

// TokenAuthenticator validates bearer tokens for RTMP connections.
type TokenAuthenticator struct {
	mu     sync.RWMutex
	tokens map[string]bool
}

// NewTokenAuthenticator creates a new token authenticator.
func NewTokenAuthenticator(tokens []string) *TokenAuthenticator {
	ta := &TokenAuthenticator{
		tokens: make(map[string]bool),
	}
	for _, token := range tokens {
		if token != "" {
			ta.tokens[token] = true
		}
	}
	return ta
}

// Authenticate checks if a token is valid.
// Returns nil if token is valid, error otherwise.
func (t *TokenAuthenticator) Authenticate(token string) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.tokens[token] {
		return fmt.Errorf("invalid token")
	}

	return nil
}

// AddToken adds a new valid token.
func (t *TokenAuthenticator) AddToken(token string) {
	if token == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.tokens[token] = true
}

// RemoveToken removes a token from valid tokens.
func (t *TokenAuthenticator) RemoveToken(token string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.tokens, token)
}

// ValidTokenCount returns the number of valid tokens.
func (t *TokenAuthenticator) ValidTokenCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.tokens)
}

// ExtractTokenFromHeader extracts bearer token from RTMP custom header format
// Format: "Bearer <token>" or just the token directly
func ExtractTokenFromHeader(header string) string {
	header = strings.TrimSpace(header)

	if header == "" {
		return ""
	}

	// Try Bearer format
	if strings.HasPrefix(header, "Bearer ") {
		token := strings.TrimSpace(header[7:])
		return token
	}

	// Return the whole thing if it doesn't start with Bearer
	return header
}
