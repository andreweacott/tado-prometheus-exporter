package auth

import (
	"github.com/clambin/tado/v2"
)

// TadoClientWrapper provides a simple interface to the Tado API client
// For now, this is a minimal wrapper, but can be extended with additional functionality
type TadoClientWrapper struct {
	client *tado.ClientWithResponses
}

// NewTadoClientWrapper creates a wrapper around a Tado client
func NewTadoClientWrapper(client *tado.ClientWithResponses) *TadoClientWrapper {
	return &TadoClientWrapper{
		client: client,
	}
}

// GetClient returns the underlying Tado client
func (tcw *TadoClientWrapper) GetClient() *tado.ClientWithResponses {
	return tcw.client
}

// Close cleanly closes the client (currently a no-op but for future extensibility)
func (tcw *TadoClientWrapper) Close() error {
	return nil
}
