package auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// MockTokenSource mocks the oauth2.TokenSource to track Token() calls
type MockTokenSource struct {
	tokenCalls int
	token      *oauth2.Token
	err        error
}

func (m *MockTokenSource) Token() (*oauth2.Token, error) {
	m.tokenCalls++
	return m.token, m.err
}

// TestPersistToken_Success verifies that persistToken successfully calls Token()
// to trigger token persistence
func TestPersistToken_Success(t *testing.T) {
	testToken := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	mockTokenSource := &MockTokenSource{
		token: testToken,
		err:   nil,
	}

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: mockTokenSource,
		},
	}

	// Call persistToken which should call Token() on the token source
	err := persistToken(client)

	assert.NoError(t, err)
	assert.Equal(t, 1, mockTokenSource.tokenCalls, "Token() should be called exactly once")
}

// TestPersistToken_TokenError verifies that persistToken returns errors from Token()
func TestPersistToken_TokenError(t *testing.T) {
	mockTokenSource := &MockTokenSource{
		token: nil,
		err:   errors.New("token retrieval failed"),
	}

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: mockTokenSource,
		},
	}

	err := persistToken(client)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token retrieval failed")
	assert.Equal(t, 1, mockTokenSource.tokenCalls)
}

// TestPersistToken_InvalidTransport verifies that persistToken rejects non-oauth2.Transport
func TestPersistToken_InvalidTransport(t *testing.T) {
	// Use a basic http.Transport instead of oauth2.Transport
	client := &http.Client{
		Transport: &http.Transport{},
	}

	err := persistToken(client)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transport type")
}

// TestPersistToken_NilTransport verifies that persistToken handles nil transport
func TestPersistToken_NilTransport(t *testing.T) {
	client := &http.Client{
		Transport: nil,
	}

	err := persistToken(client)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transport type")
}
