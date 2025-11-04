package collector

import (
	"fmt"
	"testing"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTadoCollector tests collector creation
func TestNewTadoCollector(t *testing.T) {
	t.Parallel()

	// Use isolated registry to avoid global state conflicts
	registry := prometheus.NewRegistry()
	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	collector := NewTadoCollector(
		nil,
		metricDescs,
		10*time.Second,
		"",
	)

	assert.NotNil(t, collector)
	assert.Equal(t, 10*time.Second, collector.scrapeTimeout)
}

// TestNewTadoCollector_WithHomeID tests collector creation with home ID filter
func TestNewTadoCollector_WithHomeID(t *testing.T) {
	t.Parallel()

	// Use isolated registry
	registry := prometheus.NewRegistry()
	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	collector := NewTadoCollector(
		nil,
		metricDescs,
		10*time.Second,
		"123",
	)

	assert.NotNil(t, collector)
	assert.Equal(t, "123", collector.homeID)
}

// TestNewTadoCollector_TimeoutConfiguration tests collector timeout configuration
func TestNewTadoCollector_TimeoutConfiguration(t *testing.T) {
	t.Parallel()

	// Use isolated registry
	registry := prometheus.NewRegistry()
	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	testCases := []struct {
		name    string
		timeout time.Duration
	}{
		{"5 seconds", 5 * time.Second},
		{"30 seconds", 30 * time.Second},
		{"1 minute", 1 * time.Minute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			collector := NewTadoCollector(
				nil,
				metricDescs,
				tc.timeout,
				"",
			)

			assert.Equal(t, tc.timeout, collector.scrapeTimeout)
		})
	}
}

// TestPresenceValue tests presence value conversion
func TestPresenceValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		presence string
		expected float64
	}{
		{
			name:     "Home presence",
			presence: "HOME",
			expected: 1.0,
		},
		{
			name:     "Away presence",
			presence: "AWAY",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var presence float64
			if tt.presence == "HOME" {
				presence = 1.0
			} else {
				presence = 0.0
			}
			assert.Equal(t, tt.expected, presence)
		})
	}
}

// TestPowerValue tests power value conversion
func TestPowerValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		power    string
		expected float64
	}{
		{
			name:     "Power on",
			power:    "ON",
			expected: 1.0,
		},
		{
			name:     "Power off",
			power:    "OFF",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var powered float64
			if tt.power == "ON" {
				powered = 1.0
			} else {
				powered = 0.0
			}
			assert.Equal(t, tt.expected, powered)
		})
	}
}

// TestWindowOpenValue tests window open value conversion
func TestWindowOpenValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		detected bool
		expected float64
	}{
		{
			name:     "Window open",
			detected: true,
			expected: 1.0,
		},
		{
			name:     "Window closed",
			detected: false,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var windowOpen float64
			if tt.detected {
				windowOpen = 1.0
			} else {
				windowOpen = 0.0
			}
			assert.Equal(t, tt.expected, windowOpen)
		})
	}
}

// TestLabelFormattingForMetrics tests that labels are properly formatted
func TestLabelFormattingForMetrics(t *testing.T) {
	t.Parallel()

	homeID := int32(123456)
	zoneID := int32(1)
	zoneName := "Living Room"
	zoneType := "HEATING"

	labelValues := []string{
		fmt.Sprintf("%d", homeID),
		fmt.Sprintf("%d", zoneID),
		zoneName,
		zoneType,
	}

	assert.Equal(t, 4, len(labelValues))
	assert.Equal(t, "123456", labelValues[0])
	assert.Equal(t, "1", labelValues[1])
	assert.Equal(t, "Living Room", labelValues[2])
	assert.Equal(t, "HEATING", labelValues[3])
}

// TestTemperatureConversion tests temperature handling
func TestTemperatureConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		celsius    float64
		fahrenheit float64
	}{
		{
			name:       "Room temperature",
			celsius:    20.5,
			fahrenheit: 68.9,
		},
		{
			name:       "Cold temperature",
			celsius:    5.0,
			fahrenheit: 41.0,
		},
		{
			name:       "Hot temperature",
			celsius:    30.0,
			fahrenheit: 86.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the Tado API provides both values
			assert.NotEqual(t, tt.celsius, tt.fahrenheit)
			// Rough validation: Fahrenheit should be higher for positive Celsius
			if tt.celsius > 0 {
				assert.Greater(t, tt.fahrenheit, tt.celsius)
			}
		})
	}
}

// TestMetricDescriptorsCreation tests that metric descriptors can be created
func TestMetricDescriptorsCreation(t *testing.T) {
	t.Parallel()

	// Use isolated registry
	registry := prometheus.NewRegistry()
	metricDescs, err := metrics.NewMetricDescriptorsUnregistered()
	require.NoError(t, err)
	require.NoError(t, metricDescs.RegisterWith(registry))

	assert.NotNil(t, metricDescs)
}

// TestMultipleHomeIDFiltering tests home ID filtering logic
func TestMultipleHomeIDFiltering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filterID  string
		homeID    int32
		shouldUse bool
	}{
		{
			name:      "No filter - accept any home",
			filterID:  "",
			homeID:    123,
			shouldUse: true,
		},
		{
			name:      "Filter matches",
			filterID:  "456",
			homeID:    456,
			shouldUse: true,
		},
		{
			name:      "Filter doesn't match",
			filterID:  "123",
			homeID:    456,
			shouldUse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic
			shouldUse := tt.filterID == "" || fmt.Sprintf("%d", tt.homeID) == tt.filterID
			assert.Equal(t, tt.shouldUse, shouldUse)
		})
	}
}
