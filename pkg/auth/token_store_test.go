package auth

import (
	"testing"
)

// TestTokenStorage tests that token storage is handled by clambin/tado
// The NewOAuth2Client function from clambin/tado handles all token storage:
// - Loading existing encrypted tokens from disk
// - Saving encrypted tokens with passphrase
// - Token refresh and renewal
func TestTokenStorage(t *testing.T) {
	t.Skip("Token storage is now handled internally by clambin/tado/v2 library")
}
