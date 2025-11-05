// Package auth handles OAuth2 authentication with the Tado API.
//
// It provides functions to:
//   - Authenticate users via OAuth2 device code flow
//   - Store encrypted tokens on disk
//   - Create authenticated Tado API clients
//
// The package uses the clambin/tado/v2 library which handles token encryption
// and automatic refresh. Users are guided through the authentication flow
// with a verification URL on first run.
package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/clambin/tado/v2"
	"golang.org/x/oauth2"
)

// CreateTadoClient creates a Tado API client with encrypted token storage
// On first run, it will perform OAuth device code authentication
// The user will be prompted to visit a verification URL
func CreateTadoClient(ctx context.Context, tokenPath, tokenPassphrase string) (*http.Client, error) {
	// NewOAuth2Client handles:
	// - Loading existing token from tokenPath if valid
	// - Performing device code OAuth flow if no valid token
	// - Storing encrypted token to tokenPath with tokenPassphrase
	// - Automatically refreshing token when needed
	client, err := tado.NewOAuth2Client(
		ctx,
		tokenPath,
		tokenPassphrase,
		func(response *oauth2.DeviceAuthResponse) {
			fmt.Printf("\nNo token found. Visit this link to authenticate:\n")
			fmt.Printf("%s\n\n", response.VerificationURIComplete)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 client: %w", err)
	}

	return client, nil
}

// CreateTadoClientWithHTTPClient creates a Tado API client using clambin/tado library
// This is the primary entry point for creating an authenticated Tado client
func NewAuthenticatedTadoClient(ctx context.Context, tokenPath, tokenPassphrase string) (*tado.ClientWithResponses, error) {
	httpClient, err := CreateTadoClient(ctx, tokenPath, tokenPassphrase)
	if err != nil {
		return nil, err
	}

	// Create the Tado client with the authenticated HTTP client
	client, err := tado.NewClientWithResponses(
		tado.ServerURL,
		tado.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tado client: %w", err)
	}

	return client, nil
}
