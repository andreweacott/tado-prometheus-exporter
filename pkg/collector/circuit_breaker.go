package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/clambin/tado/v2"
	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig configures the circuit breaker behavior
type CircuitBreakerConfig struct {
	// MaxConsecutiveFailures is the number of consecutive failures before opening
	MaxConsecutiveFailures uint32
	// Timeout is how long the circuit breaker stays open before trying half-open
	Timeout time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxConsecutiveFailures: 5,
		Timeout:                30 * time.Second,
	}
}

// circuitBreakerAPI wraps TadoAPI with circuit breaker protection
type circuitBreakerAPI struct {
	api      TadoAPI
	breaker  *gobreaker.CircuitBreaker
	timeout  time.Duration
	state    CircuitBreakerState
	lastErr  error
	lastTime time.Time
}

// CircuitBreakerState represents the circuit breaker state
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewTadoAPIWithCircuitBreaker wraps a TadoAPI with circuit breaker protection
func NewTadoAPIWithCircuitBreaker(api TadoAPI, config CircuitBreakerConfig) TadoAPI {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "TadoAPI",
		MaxRequests: 1,
		Interval:    config.Timeout,
		Timeout:     2 * config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= config.MaxConsecutiveFailures
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log state changes
			// Could also update metrics here
		},
	})

	return &circuitBreakerAPI{
		api:     api,
		breaker: cb,
		timeout: config.Timeout,
		state:   CircuitClosed,
	}
}

// GetMe implements TadoAPI.GetMe with circuit breaker protection
func (cb *circuitBreakerAPI) GetMe(ctx context.Context) (*tado.User, error) {
	result, err := cb.breaker.Execute(func() (interface{}, error) {
		return cb.api.GetMe(ctx)
	})

	if err != nil {
		cb.lastErr = err
		cb.lastTime = time.Now()
		return nil, cb.wrapError(err)
	}

	return result.(*tado.User), nil
}

// GetHomeState implements TadoAPI.GetHomeState with circuit breaker protection
func (cb *circuitBreakerAPI) GetHomeState(ctx context.Context, homeID tado.HomeId) (*tado.HomeState, error) {
	result, err := cb.breaker.Execute(func() (interface{}, error) {
		return cb.api.GetHomeState(ctx, homeID)
	})

	if err != nil {
		cb.lastErr = err
		cb.lastTime = time.Now()
		return nil, cb.wrapError(err)
	}

	return result.(*tado.HomeState), nil
}

// GetZones implements TadoAPI.GetZones with circuit breaker protection
func (cb *circuitBreakerAPI) GetZones(ctx context.Context, homeID tado.HomeId) ([]tado.Zone, error) {
	result, err := cb.breaker.Execute(func() (interface{}, error) {
		return cb.api.GetZones(ctx, homeID)
	})

	if err != nil {
		cb.lastErr = err
		cb.lastTime = time.Now()
		return nil, cb.wrapError(err)
	}

	return result.([]tado.Zone), nil
}

// GetZoneStates implements TadoAPI.GetZoneStates with circuit breaker protection
func (cb *circuitBreakerAPI) GetZoneStates(ctx context.Context, homeID tado.HomeId) (*tado.ZoneStates, error) {
	result, err := cb.breaker.Execute(func() (interface{}, error) {
		return cb.api.GetZoneStates(ctx, homeID)
	})

	if err != nil {
		cb.lastErr = err
		cb.lastTime = time.Now()
		return nil, cb.wrapError(err)
	}

	return result.(*tado.ZoneStates), nil
}

// GetWeather implements TadoAPI.GetWeather with circuit breaker protection
func (cb *circuitBreakerAPI) GetWeather(ctx context.Context, homeID tado.HomeId) (*tado.Weather, error) {
	result, err := cb.breaker.Execute(func() (interface{}, error) {
		return cb.api.GetWeather(ctx, homeID)
	})

	if err != nil {
		cb.lastErr = err
		cb.lastTime = time.Now()
		return nil, cb.wrapError(err)
	}

	return result.(*tado.Weather), nil
}

// wrapError converts circuit breaker errors to user-friendly messages
func (cb *circuitBreakerAPI) wrapError(err error) error {
	if err == gobreaker.ErrOpenState {
		cb.state = CircuitOpen
		return fmt.Errorf("circuit breaker is open: API is temporarily unavailable (will retry after %v)", cb.timeout)
	}

	if err == gobreaker.ErrTooManyRequests {
		cb.state = CircuitHalfOpen
		return fmt.Errorf("circuit breaker is half-open: testing API recovery")
	}

	cb.state = CircuitClosed
	return err
}

// State returns the current circuit breaker state
func (cb *circuitBreakerAPI) State() CircuitBreakerState {
	switch cb.breaker.State() {
	case gobreaker.StateClosed:
		return CircuitClosed
	case gobreaker.StateOpen:
		return CircuitOpen
	case gobreaker.StateHalfOpen:
		return CircuitHalfOpen
	default:
		return CircuitClosed
	}
}

// LastError returns the last error that occurred
func (cb *circuitBreakerAPI) LastError() error {
	return cb.lastErr
}

// LastErrorTime returns when the last error occurred
func (cb *circuitBreakerAPI) LastErrorTime() time.Time {
	return cb.lastTime
}
