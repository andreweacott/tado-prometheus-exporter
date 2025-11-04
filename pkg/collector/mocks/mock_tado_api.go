// Package mocks provides test doubles for collector package.
package mocks

import (
	"context"

	"github.com/clambin/tado/v2"
	"github.com/stretchr/testify/mock"
)

// MockTadoAPI is a mock implementation of the TadoAPI interface
type MockTadoAPI struct {
	mock.Mock
}

// GetMe implements TadoAPI.GetMe
func (m *MockTadoAPI) GetMe(ctx context.Context) (*tado.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tado.User), args.Error(1)
}

// GetHomeState implements TadoAPI.GetHomeState
func (m *MockTadoAPI) GetHomeState(ctx context.Context, homeID tado.HomeId) (*tado.HomeState, error) {
	args := m.Called(ctx, homeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tado.HomeState), args.Error(1)
}

// GetZones implements TadoAPI.GetZones
func (m *MockTadoAPI) GetZones(ctx context.Context, homeID tado.HomeId) ([]tado.Zone, error) {
	args := m.Called(ctx, homeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tado.Zone), args.Error(1)
}

// GetZoneStates implements TadoAPI.GetZoneStates
func (m *MockTadoAPI) GetZoneStates(ctx context.Context, homeID tado.HomeId) (*tado.ZoneStates, error) {
	args := m.Called(ctx, homeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tado.ZoneStates), args.Error(1)
}

// GetWeather implements TadoAPI.GetWeather
func (m *MockTadoAPI) GetWeather(ctx context.Context, homeID tado.HomeId) (*tado.Weather, error) {
	args := m.Called(ctx, homeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tado.Weather), args.Error(1)
}

// ExpectGetMeReturnsHomes sets up expectation for GetMe to return homes
func (m *MockTadoAPI) ExpectGetMeReturnsHomes(homeIDs []tado.HomeId) *MockTadoAPI {
	homes := make([]tado.HomeBase, len(homeIDs))
	for i, id := range homeIDs {
		homes[i] = tado.HomeBase{Id: &id}
	}
	m.On("GetMe", mock.Anything).Return(&tado.User{Homes: &homes}, nil)
	return m
}

// ExpectGetMeReturnsError sets up expectation for GetMe to return an error
func (m *MockTadoAPI) ExpectGetMeReturnsError(err error) *MockTadoAPI {
	m.On("GetMe", mock.Anything).Return(nil, err)
	return m
}

// ExpectGetMeReturnsEmptyHomes sets up expectation for GetMe to return no homes
func (m *MockTadoAPI) ExpectGetMeReturnsEmptyHomes() *MockTadoAPI {
	emptyHomes := []tado.HomeBase{}
	m.On("GetMe", mock.Anything).Return(&tado.User{Homes: &emptyHomes}, nil)
	return m
}

// ExpectAllAPICalls sets up default expectations for all API calls
func (m *MockTadoAPI) ExpectAllAPICalls() *MockTadoAPI {
	// Default: return empty but valid responses
	emptyHomes := []tado.HomeBase{}
	m.On("GetMe", mock.Anything).Return(&tado.User{Homes: &emptyHomes}, nil)
	m.On("GetHomeState", mock.Anything, mock.Anything).Return(&tado.HomeState{}, nil)
	m.On("GetZones", mock.Anything, mock.Anything).Return([]tado.Zone{}, nil)
	emptyZoneStates := map[string]tado.ZoneState{}
	m.On("GetZoneStates", mock.Anything, mock.Anything).Return(&tado.ZoneStates{ZoneStates: &emptyZoneStates}, nil)
	m.On("GetWeather", mock.Anything, mock.Anything).Return(&tado.Weather{}, nil)
	return m
}
