package auth

import (
	"testing"
)

func TestNewTokenAuthenticator(t *testing.T) {
	tokens := []string{"token1", "token2", "token3"}
	auth := NewTokenAuthenticator(tokens)

	if auth == nil {
		t.Error("NewTokenAuthenticator returned nil")
	}
	if auth.ValidTokenCount() != 3 {
		t.Errorf("ValidTokenCount = %d, want 3", auth.ValidTokenCount())
	}
}

func TestAuthenticateValid(t *testing.T) {
	auth := NewTokenAuthenticator([]string{"secret-token"})

	err := auth.Authenticate("secret-token")
	if err != nil {
		t.Errorf("Authenticate valid token failed: %v", err)
	}
}

func TestAuthenticateInvalid(t *testing.T) {
	auth := NewTokenAuthenticator([]string{"secret-token"})

	tests := []struct {
		name  string
		token string
	}{
		{"invalid token", "wrong-token"},
		{"empty token", ""},
		{"nil equivalent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Authenticate(tt.token)
			if err == nil {
				t.Error("Authenticate should have failed for invalid token")
			}
		})
	}
}

func TestAddToken(t *testing.T) {
	auth := NewTokenAuthenticator([]string{"token1"})

	if auth.ValidTokenCount() != 1 {
		t.Errorf("Initial count = %d, want 1", auth.ValidTokenCount())
	}

	auth.AddToken("token2")
	if auth.ValidTokenCount() != 2 {
		t.Errorf("After add count = %d, want 2", auth.ValidTokenCount())
	}

	// Verify new token works
	err := auth.Authenticate("token2")
	if err != nil {
		t.Errorf("New token authentication failed: %v", err)
	}
}

func TestRemoveToken(t *testing.T) {
	auth := NewTokenAuthenticator([]string{"token1", "token2"})

	if auth.ValidTokenCount() != 2 {
		t.Errorf("Initial count = %d, want 2", auth.ValidTokenCount())
	}

	auth.RemoveToken("token1")
	if auth.ValidTokenCount() != 1 {
		t.Errorf("After remove count = %d, want 1", auth.ValidTokenCount())
	}

	// Verify removed token fails
	err := auth.Authenticate("token1")
	if err == nil {
		t.Error("Removed token should fail authentication")
	}

	// Verify remaining token works
	err = auth.Authenticate("token2")
	if err != nil {
		t.Errorf("Remaining token authentication failed: %v", err)
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"Bearer format", "Bearer secret-token", "secret-token"},
		{"Bearer with extra spaces", "Bearer   secret-token", "secret-token"},
		{"Plain token", "secret-token", "secret-token"},
		{"Empty", "", ""},
		{"Only Bearer word", "Bearer", "Bearer"}, // Returns the word itself when no space
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTokenFromHeader(tt.header)
			if got != tt.want {
				t.Errorf("ExtractTokenFromHeader(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestConcurrentAuthenticate(t *testing.T) {
	auth := NewTokenAuthenticator([]string{"token1", "token2", "token3"})

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(tokenNum int) {
			tokens := []string{"token1", "token2", "token3"}
			token := tokens[tokenNum%3]
			auth.Authenticate(token)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEmptyTokensIgnored(t *testing.T) {
	auth := NewTokenAuthenticator([]string{"", "token1", "", "token2"})

	if auth.ValidTokenCount() != 2 {
		t.Errorf("Empty tokens should be ignored, got count %d, want 2", auth.ValidTokenCount())
	}
}
