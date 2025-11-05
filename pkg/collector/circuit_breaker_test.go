package collector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/collector/mocks"
	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreakerStartsClosed tests that circuit breaker starts in closed state
func TestCircuitBreakerStartsClosed(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}
	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 3,
		Timeout:                10 * time.Millisecond,
	})

	cbAPI, ok := cb.(*circuitBreakerAPI)
	require.True(t, ok)
	assert.Equal(t, CircuitClosed, cbAPI.State())
}

// TestCircuitBreakerOpensOnFailures tests circuit breaker opens after consecutive failures
func TestCircuitBreakerOpensOnFailures(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("API error"))

	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
		Timeout:                100 * time.Millisecond,
	})

	ctx := context.Background()

	// First failure
	_, err := cb.GetMe(ctx)
	require.Error(t, err)

	// Second failure - should open
	_, err = cb.GetMe(ctx)
	require.Error(t, err)

	// Next call should fail immediately with ErrOpenState
	_, err = cb.GetMe(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

// TestCircuitBreakerRecovery tests circuit breaker recovers after timeout
func TestCircuitBreakerRecovery(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}

	// First 2 calls fail, subsequent calls succeed
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("API error")).Twice()
	mockAPI.On("GetMe", mock.Anything).Return(&tado.User{}, nil)

	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
		Timeout:                50 * time.Millisecond,
	})

	ctx := context.Background()

	// Cause failures to open circuit
	_, _ = cb.GetMe(ctx)
	_, _ = cb.GetMe(ctx)

	cbAPI, ok := cb.(*circuitBreakerAPI)
	require.True(t, ok)
	assert.Equal(t, CircuitOpen, cbAPI.State())

	// Wait for half-open timeout
	time.Sleep(100 * time.Millisecond)

	// Next call should test recovery
	_, err := cb.GetMe(ctx)
	require.NoError(t, err)

	// Should be closed after success
	assert.Equal(t, CircuitClosed, cbAPI.State())
}

// TestCircuitBreakerSuccessResetsCount tests that successful calls reset the error count
func TestCircuitBreakerSuccessResetsCount(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}

	// Mix of success and failures
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("error")).Once()
	mockAPI.On("GetMe", mock.Anything).Return(&tado.User{}, nil).Once()
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("error")).Once()
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("error")).Once()

	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 3,
		Timeout:                100 * time.Millisecond,
	})

	ctx := context.Background()

	// Failure, success, then 2 failures - shouldn't open yet because count reset on success
	_, _ = cb.GetMe(ctx)
	_, _ = cb.GetMe(ctx)
	_, _ = cb.GetMe(ctx)
	_, _ = cb.GetMe(ctx)

	cbAPI, ok := cb.(*circuitBreakerAPI)
	require.True(t, ok)
	assert.Equal(t, CircuitClosed, cbAPI.State())
}

// TestCircuitBreakerAllMethods tests circuit breaker protects all API methods
func TestCircuitBreakerAllMethods(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("error"))
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))
	mockAPI.On("GetZones", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))
	mockAPI.On("GetZoneStates", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))
	mockAPI.On("GetWeather", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))

	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 1,
		Timeout:                100 * time.Millisecond,
	})

	ctx := context.Background()
	homeID := tado.HomeId(123)

	// Fail one method
	_, err := cb.GetMe(ctx)
	require.Error(t, err)

	// Circuit should be open for all methods
	_, err = cb.GetHomeState(ctx, homeID)
	assert.Error(t, err)

	_, err = cb.GetZones(ctx, homeID)
	assert.Error(t, err)

	_, err = cb.GetZoneStates(ctx, homeID)
	assert.Error(t, err)

	_, err = cb.GetWeather(ctx, homeID)
	assert.Error(t, err)
}

// TestCircuitBreakerErrorTracking tests that errors are tracked
func TestCircuitBreakerErrorTracking(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.On("GetMe", mock.Anything).Return(nil, fmt.Errorf("test error"))

	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 5,
		Timeout:                100 * time.Millisecond,
	})

	ctx := context.Background()

	// Should have no error initially
	cbAPI, ok := cb.(*circuitBreakerAPI)
	require.True(t, ok)
	assert.Nil(t, cbAPI.LastError())

	// Cause an error
	startTime := time.Now()
	_, _ = cb.GetMe(ctx)

	// Error should be tracked
	assert.NotNil(t, cbAPI.LastError())
	assert.Contains(t, cbAPI.LastError().Error(), "test error")
	assert.True(t, cbAPI.LastErrorTime().After(startTime))
}

// TestCircuitBreakerDefaultConfig tests default configuration
func TestCircuitBreakerDefaultConfig(t *testing.T) {
	t.Parallel()

	config := DefaultCircuitBreakerConfig()
	assert.Equal(t, uint32(5), config.MaxConsecutiveFailures)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

// TestCircuitBreakerPartialSuccess tests behavior on partial success
func TestCircuitBreakerPartialSuccess(t *testing.T) {
	t.Parallel()

	mockAPI := &mocks.MockTadoAPI{}

	// Simulate partial success - GetMe works, GetHomeState fails
	mockAPI.On("GetMe", mock.Anything).Return(&tado.User{}, nil)
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error"))

	cb := NewTadoAPIWithCircuitBreaker(mockAPI, CircuitBreakerConfig{
		MaxConsecutiveFailures: 5,
		Timeout:                100 * time.Millisecond,
	})

	ctx := context.Background()
	homeID := tado.HomeId(123)

	// GetMe succeeds
	user, err := cb.GetMe(ctx)
	require.NoError(t, err)
	assert.NotNil(t, user)

	// GetHomeState fails (but circuit breaker treats all methods as same)
	homeState, err := cb.GetHomeState(ctx, homeID)
	require.Error(t, err)
	assert.Nil(t, homeState)

	// Circuit is still using same breaker, so it starts counting failures
	// But we haven't reached threshold yet
	cbAPI, ok := cb.(*circuitBreakerAPI)
	require.True(t, ok)
	// After 1 failure on different method, circuit should still be closed
	assert.Equal(t, CircuitClosed, cbAPI.State())
}
