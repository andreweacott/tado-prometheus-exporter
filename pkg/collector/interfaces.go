// Package collector provides interfaces for Tado API interactions.
package collector

import (
	"context"

	"github.com/clambin/tado/v2"
)

// TadoAPI defines the interface for Tado API interactions.
// This interface allows for dependency injection and testing with mocks.
type TadoAPI interface {
	// GetMe retrieves the current user information
	GetMe(ctx context.Context) (*tado.User, error)

	// GetHomeState retrieves the state of a home (presence, etc.)
	GetHomeState(ctx context.Context, homeID tado.HomeId) (*tado.HomeState, error)

	// GetZones retrieves all zones in a home
	GetZones(ctx context.Context, homeID tado.HomeId) ([]tado.Zone, error)

	// GetZoneStates retrieves the current state of all zones in a home
	GetZoneStates(ctx context.Context, homeID tado.HomeId) (*tado.ZoneStates, error)

	// GetWeather retrieves weather information for a home
	GetWeather(ctx context.Context, homeID tado.HomeId) (*tado.Weather, error)
}
