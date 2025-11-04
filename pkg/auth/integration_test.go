package auth

import (
	"testing"
)

// TestAuthenticationIntegration documents the new authentication flow with clambin/tado
// The new authentication flow works as follows:
// 1. Call NewAuthenticatedTadoClient(ctx, tokenPath, tokenPassphrase)
// 2. If valid token exists at tokenPath (encrypted with passphrase), it's loaded
// 3. If no valid token exists, device code OAuth flow is initiated
// 4. User is prompted with verification URL
// 5. After authorization, token is saved encrypted to tokenPath
// 6. Tado client is returned ready to use
//
// This is all handled transparently by clambin/tado library
func TestAuthenticationIntegration(t *testing.T) {
	t.Skip("Integration test - requires token storage and OAuth setup")
}
