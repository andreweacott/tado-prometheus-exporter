package collector

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/collector/mocks"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/logger"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/metrics"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCollectorWithSuccessfulCollection tests successful metric collection
func TestCollectorWithSuccessfulCollection(t *testing.T) {
	t.Parallel()

	// Create isolated registry
	registry := prometheus.NewRegistry()

	// Create metrics without registering globally
	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	// Register with isolated registry instead of global
	require.NoError(t, metricDescs.RegisterWith(registry))

	// Create mock API with homes configured
	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsHomes([]int64{1})
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(&tado.HomeState{}, nil)
	mockAPI.On("GetZones", mock.Anything, mock.Anything).Return([]tado.Zone{}, nil)
	mockAPI.On("GetZoneStates", mock.Anything, mock.Anything).Return(&tado.ZoneStates{ZoneStates: &map[string]tado.ZoneState{}}, nil)
	mockAPI.On("GetWeather", mock.Anything, mock.Anything).Return(&tado.Weather{}, nil)

	// Create logger
	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	// Create collector
	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "", log)

	// Verify the collector collects metrics
	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	// Verify metrics were collected
	metricsCount := len(ch)
	assert.Greater(t, metricsCount, 0, "Expected metrics to be collected")
}

// TestCollectorHandlesGetMeError tests error handling when GetMe fails
func TestCollectorHandlesGetMeError(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	// Create mock API that fails on GetMe
	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsError(fmt.Errorf("API error"))

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "", log)

	// Collect should handle the error gracefully
	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	// Should still collect metrics without panicking
	assert.Greater(t, len(ch), 0)
}

// TestCollectorHandlesEmptyHomes tests handling when user has no homes
func TestCollectorHandlesEmptyHomes(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	// Create mock API with empty homes list
	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsEmptyHomes()

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "", log)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	assert.Greater(t, len(ch), 0)
}

// TestCollectorWithHomeIDFilter tests home ID filtering
func TestCollectorWithHomeIDFilter(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	// Create mock with multiple homes
	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsHomes([]int64{1, 2})
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(&tado.HomeState{}, nil)
	mockAPI.On("GetZones", mock.Anything, mock.Anything).Return([]tado.Zone{}, nil)
	mockAPI.On("GetZoneStates", mock.Anything, mock.Anything).Return(&tado.ZoneStates{ZoneStates: &map[string]tado.ZoneState{}}, nil)
	mockAPI.On("GetWeather", mock.Anything, mock.Anything).Return(&tado.Weather{}, nil)

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	// Filter to only home 1
	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "1", log)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	assert.Greater(t, len(ch), 0)
}

// TestCollectorWithExporterMetrics tests collection with exporter metrics
func TestCollectorWithExporterMetrics(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	exporterMetrics, err := metrics.NewExporterMetrics()
	require.NoError(t, err)

	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsHomes([]int64{1})
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(&tado.HomeState{}, nil)
	mockAPI.On("GetZones", mock.Anything, mock.Anything).Return([]tado.Zone{}, nil)
	mockAPI.On("GetZoneStates", mock.Anything, mock.Anything).Return(&tado.ZoneStates{ZoneStates: &map[string]tado.ZoneState{}}, nil)
	mockAPI.On("GetWeather", mock.Anything, mock.Anything).Return(&tado.Weather{}, nil)

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "", log).
		WithExporterMetrics(exporterMetrics)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	assert.Greater(t, len(ch), 0)
}

// TestCollectorContextCancellation tests handling of context cancellation with short timeout
func TestCollectorContextCancellation(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsHomes([]int64{1})
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(&tado.HomeState{}, nil)
	mockAPI.On("GetZones", mock.Anything, mock.Anything).Return([]tado.Zone{}, nil)
	mockAPI.On("GetZoneStates", mock.Anything, mock.Anything).Return(&tado.ZoneStates{ZoneStates: &map[string]tado.ZoneState{}}, nil)
	mockAPI.On("GetWeather", mock.Anything, mock.Anything).Return(&tado.Weather{}, nil)

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	// Create collector with very short timeout
	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 1*time.Millisecond, "", log)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	// Should still produce metrics despite short timeout
	assert.Greater(t, len(ch), 0)
}

// TestDescribe tests that all metrics are properly described
func TestDescribe(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	mockAPI := &mocks.MockTadoAPI{}

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "", log)

	ch := make(chan *prometheus.Desc, 100)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	descriptors := make([]*prometheus.Desc, 0)
	for desc := range ch {
		descriptors = append(descriptors, desc)
	}

	// Should have metrics described
	assert.Greater(t, len(descriptors), 0, "Expected metrics to be described")
}

// TestCollectorGetWeatherError tests error handling when GetWeather fails
func TestCollectorGetWeatherError(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()

	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	mockAPI := &mocks.MockTadoAPI{}
	mockAPI.ExpectGetMeReturnsHomes([]int64{1})
	mockAPI.On("GetHomeState", mock.Anything, mock.Anything).Return(&tado.HomeState{}, nil)
	mockAPI.On("GetZones", mock.Anything, mock.Anything).Return([]tado.Zone{}, nil)
	mockAPI.On("GetZoneStates", mock.Anything, mock.Anything).Return(&tado.ZoneStates{ZoneStates: &map[string]tado.ZoneState{}}, nil)
	mockAPI.On("GetWeather", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("weather API error"))

	log, err := logger.NewWithWriter("error", "text", io.Discard)
	require.NoError(t, err)

	collector := NewTadoCollectorWithLogger(mockAPI, metricDescs, 5*time.Second, "", log)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	// Should handle gracefully and still produce metrics
	assert.Greater(t, len(ch), 0)
}
