// Package collector provides adapter for existing Tado client.
package collector

import (
	"context"
	"fmt"

	"github.com/clambin/tado/v2"
)

// TadoClientAdapter adapts *tado.ClientWithResponses to implement TadoAPI interface
type TadoClientAdapter struct {
	client *tado.ClientWithResponses
}

// NewTadoClientAdapter creates a new adapter for the Tado client
func NewTadoClientAdapter(client *tado.ClientWithResponses) TadoAPI {
	return &TadoClientAdapter{client: client}
}

// GetMe implements TadoAPI.GetMe
func (a *TadoClientAdapter) GetMe(ctx context.Context) (*tado.User, error) {
	response, err := a.client.GetMeWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get me: %w", err)
	}

	if response.StatusCode() != 200 || response.JSON200 == nil {
		return nil, fmt.Errorf("failed to get me: status code %d", response.StatusCode())
	}

	return response.JSON200, nil
}

// GetHomeState implements TadoAPI.GetHomeState
func (a *TadoClientAdapter) GetHomeState(ctx context.Context, homeID tado.HomeId) (*tado.HomeState, error) {
	response, err := a.client.GetHomeStateWithResponse(ctx, homeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get home state: %w", err)
	}

	if response.StatusCode() != 200 || response.JSON200 == nil {
		return nil, fmt.Errorf("failed to get home state: status code %d", response.StatusCode())
	}

	return response.JSON200, nil
}

// GetZones implements TadoAPI.GetZones
func (a *TadoClientAdapter) GetZones(ctx context.Context, homeID tado.HomeId) ([]tado.Zone, error) {
	response, err := a.client.GetZonesWithResponse(ctx, homeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones: %w", err)
	}

	if response.StatusCode() != 200 || response.JSON200 == nil {
		return nil, fmt.Errorf("failed to get zones: status code %d", response.StatusCode())
	}

	return *response.JSON200, nil
}

// GetZoneStates implements TadoAPI.GetZoneStates
func (a *TadoClientAdapter) GetZoneStates(ctx context.Context, homeID tado.HomeId) (*tado.ZoneStates, error) {
	response, err := a.client.GetZoneStatesWithResponse(ctx, homeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone states: %w", err)
	}

	if response.StatusCode() != 200 || response.JSON200 == nil {
		return nil, fmt.Errorf("failed to get zone states: status code %d", response.StatusCode())
	}

	return response.JSON200, nil
}

// GetWeather implements TadoAPI.GetWeather
func (a *TadoClientAdapter) GetWeather(ctx context.Context, homeID tado.HomeId) (*tado.Weather, error) {
	response, err := a.client.GetWeatherWithResponse(ctx, homeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather: %w", err)
	}

	if response.StatusCode() != 200 || response.JSON200 == nil {
		return nil, fmt.Errorf("failed to get weather: status code %d", response.StatusCode())
	}

	return response.JSON200, nil
}
