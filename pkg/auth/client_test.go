package auth

import (
	"testing"

	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
)

// TestTadoClientWrapper tests the Tado client wrapper
func TestTadoClientWrapper(t *testing.T) {
	// Since we're using clambin/tado now, wrapper is minimal
	// Just test that it can be created and doesn't error
	wrapper := NewTadoClientWrapper(nil)
	assert.NotNil(t, wrapper)

	// Test GetClient returns the same client we put in
	client, err := tado.NewClientWithResponses("https://test.example.com")
	if err == nil {
		wrapper = NewTadoClientWrapper(client)
		assert.Equal(t, client, wrapper.GetClient())
	}
}

// TestTadoClientWrapperClose tests the Close method
func TestTadoClientWrapperClose(t *testing.T) {
	wrapper := NewTadoClientWrapper(nil)
	err := wrapper.Close()
	assert.NoError(t, err)
}
