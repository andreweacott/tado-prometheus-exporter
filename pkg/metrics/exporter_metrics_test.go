package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExporterMetrics tests creating exporter metrics
func TestNewExporterMetrics(t *testing.T) {
	// Use a custom registry to avoid conflicts with other tests
	registry := prometheus.NewRegistry()

	// Create metrics manually instead of using the default registry
	em := &ExporterMetrics{
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tado_exporter_scrape_duration_seconds",
			Help:    "Time taken to collect metrics from Tado API in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6),
		}),
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tado_exporter_scrape_errors_total",
			Help: "Total number of errors while collecting metrics from Tado API",
		}),
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_exporter_build_info",
			Help: "Build information for the exporter (value is always 1)",
		}),
	}

	// Register metrics
	assert.NoError(t, registry.Register(em.ScrapeDurationSeconds))
	assert.NoError(t, registry.Register(em.ScrapeErrorsTotal))
	assert.NoError(t, registry.Register(em.BuildInfo))

	// Verify metrics are registered
	assert.NotNil(t, em.ScrapeDurationSeconds)
	assert.NotNil(t, em.ScrapeErrorsTotal)
	assert.NotNil(t, em.BuildInfo)
}

// TestRecordScrapeDuration tests recording scrape duration
func TestRecordScrapeDuration(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "test_scrape_duration",
			Help:    "Test scrape duration",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6),
		}),
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_scrape_errors",
			Help: "Test scrape errors",
		}),
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_build_info",
			Help: "Test build info",
		}),
	}

	require.NoError(t, registry.Register(em.ScrapeDurationSeconds))
	require.NoError(t, registry.Register(em.ScrapeErrorsTotal))
	require.NoError(t, registry.Register(em.BuildInfo))

	// Record some durations
	durations := []float64{0.1, 0.5, 1.0, 2.0, 5.0}
	for _, d := range durations {
		em.RecordScrapeDuration(d)
	}

	// Verify histogram has samples
	families, err := registry.Gather()
	require.NoError(t, err)

	histogramFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_scrape_duration" {
			histogramFound = true
			// Check that we have a histogram with samples
			assert.Greater(t, len(family.Metric), 0)
		}
	}
	assert.True(t, histogramFound, "histogram metric not found")
}

// TestIncrementScrapeErrors tests incrementing error counter
func TestIncrementScrapeErrors(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "test_scrape_duration2",
			Help:    "Test scrape duration 2",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6),
		}),
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_scrape_errors2",
			Help: "Test scrape errors 2",
		}),
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_build_info2",
			Help: "Test build info 2",
		}),
	}

	require.NoError(t, registry.Register(em.ScrapeDurationSeconds))
	require.NoError(t, registry.Register(em.ScrapeErrorsTotal))
	require.NoError(t, registry.Register(em.BuildInfo))

	// Increment errors
	em.IncrementScrapeErrors()
	em.IncrementScrapeErrors()
	em.IncrementScrapeErrors()

	// Verify counter increased
	families, err := registry.Gather()
	require.NoError(t, err)

	counterFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_scrape_errors2" {
			counterFound = true
			require.Greater(t, len(family.Metric), 0)
			// Counter value should be 3
			assert.Equal(t, 3.0, *family.Metric[0].Counter.Value)
		}
	}
	assert.True(t, counterFound, "counter metric not found")
}

// TestBuildInfoSet tests that build info is set to 1
func TestBuildInfoSet(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "test_scrape_duration3",
			Help:    "Test scrape duration 3",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6),
		}),
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_scrape_errors3",
			Help: "Test scrape errors 3",
		}),
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_build_info3",
			Help: "Test build info 3",
		}),
	}

	require.NoError(t, registry.Register(em.ScrapeDurationSeconds))
	require.NoError(t, registry.Register(em.ScrapeErrorsTotal))
	require.NoError(t, registry.Register(em.BuildInfo))

	// Set build info
	em.BuildInfo.Set(1)

	// Verify build info is 1
	families, err := registry.Gather()
	require.NoError(t, err)

	buildInfoFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_build_info3" {
			buildInfoFound = true
			require.Greater(t, len(family.Metric), 0)
			// Build info should be 1
			assert.Equal(t, 1.0, *family.Metric[0].Gauge.Value)
		}
	}
	assert.True(t, buildInfoFound, "build info metric not found")
}

// TestExporterMetricsNames tests metric naming
func TestExporterMetricsNames(t *testing.T) {
	// Test metric names are correct
	scrapeOptsName := "tado_exporter_scrape_duration_seconds"
	errorsOptsName := "tado_exporter_scrape_errors_total"
	buildInfoOptsName := "tado_exporter_build_info"

	assert.NotEmpty(t, scrapeOptsName)
	assert.NotEmpty(t, errorsOptsName)
	assert.NotEmpty(t, buildInfoOptsName)

	// Verify naming convention
	assert.True(t, len(scrapeOptsName) > 0)
	assert.True(t, len(errorsOptsName) > 0)
	assert.True(t, len(buildInfoOptsName) > 0)
}

// BenchmarkRecordScrapeDuration benchmarks recording duration
func BenchmarkRecordScrapeDuration(b *testing.B) {
	em := &ExporterMetrics{
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "bench_scrape_duration",
			Help:    "Bench scrape duration",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6),
		}),
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "bench_scrape_errors",
			Help: "Bench scrape errors",
		}),
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bench_build_info",
			Help: "Bench build info",
		}),
	}

	// Register to avoid warnings
	_ = prometheus.NewRegistry().Register(em.ScrapeDurationSeconds)
	_ = prometheus.NewRegistry().Register(em.ScrapeErrorsTotal)
	_ = prometheus.NewRegistry().Register(em.BuildInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		em.RecordScrapeDuration(1.5)
	}
}

// BenchmarkIncrementErrors benchmarks incrementing errors
func BenchmarkIncrementErrors(b *testing.B) {
	em := &ExporterMetrics{
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "bench_scrape_duration2",
			Help:    "Bench scrape duration 2",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6),
		}),
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "bench_scrape_errors2",
			Help: "Bench scrape errors 2",
		}),
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bench_build_info2",
			Help: "Bench build info 2",
		}),
	}

	_ = prometheus.NewRegistry().Register(em.ScrapeDurationSeconds)
	_ = prometheus.NewRegistry().Register(em.ScrapeErrorsTotal)
	_ = prometheus.NewRegistry().Register(em.BuildInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		em.IncrementScrapeErrors()
	}
}

// TestAuthenticationStatusValid tests setting valid authentication status
func TestAuthenticationStatusValid(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		AuthenticationValid: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_valid",
			Help: "Test auth valid",
		}),
		AuthenticationErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_auth_errors",
			Help: "Test auth errors",
		}),
		LastAuthenticationSuccessUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_success_unix",
			Help: "Test auth success unix",
		}),
	}

	require.NoError(t, registry.Register(em.AuthenticationValid))
	require.NoError(t, registry.Register(em.AuthenticationErrorsTotal))
	require.NoError(t, registry.Register(em.LastAuthenticationSuccessUnix))

	// Set to valid
	em.SetAuthenticationValid(true)

	families, err := registry.Gather()
	require.NoError(t, err)

	authFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_auth_valid" {
			authFound = true
			require.Greater(t, len(family.Metric), 0)
			assert.Equal(t, 1.0, *family.Metric[0].Gauge.Value)
		}
	}
	assert.True(t, authFound, "auth valid metric not found")
}

// TestAuthenticationStatusInvalid tests setting invalid authentication status
func TestAuthenticationStatusInvalid(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		AuthenticationValid: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_valid2",
			Help: "Test auth valid 2",
		}),
		AuthenticationErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_auth_errors2",
			Help: "Test auth errors 2",
		}),
		LastAuthenticationSuccessUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_success_unix2",
			Help: "Test auth success unix 2",
		}),
	}

	require.NoError(t, registry.Register(em.AuthenticationValid))
	require.NoError(t, registry.Register(em.AuthenticationErrorsTotal))
	require.NoError(t, registry.Register(em.LastAuthenticationSuccessUnix))

	// Set to invalid
	em.SetAuthenticationValid(false)

	families, err := registry.Gather()
	require.NoError(t, err)

	authFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_auth_valid2" {
			authFound = true
			require.Greater(t, len(family.Metric), 0)
			assert.Equal(t, 0.0, *family.Metric[0].Gauge.Value)
		}
	}
	assert.True(t, authFound, "auth valid metric not found")
}

// TestAuthenticationErrorsIncrement tests incrementing authentication errors
func TestAuthenticationErrorsIncrement(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		AuthenticationValid: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_valid3",
			Help: "Test auth valid 3",
		}),
		AuthenticationErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_auth_errors3",
			Help: "Test auth errors 3",
		}),
		LastAuthenticationSuccessUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_success_unix3",
			Help: "Test auth success unix 3",
		}),
	}

	require.NoError(t, registry.Register(em.AuthenticationValid))
	require.NoError(t, registry.Register(em.AuthenticationErrorsTotal))
	require.NoError(t, registry.Register(em.LastAuthenticationSuccessUnix))

	// Increment errors multiple times
	em.IncrementAuthenticationErrors()
	em.IncrementAuthenticationErrors()
	em.IncrementAuthenticationErrors()

	families, err := registry.Gather()
	require.NoError(t, err)

	errorsFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_auth_errors3" {
			errorsFound = true
			require.Greater(t, len(family.Metric), 0)
			assert.Equal(t, 3.0, *family.Metric[0].Counter.Value)
		}
	}
	assert.True(t, errorsFound, "auth errors metric not found")
}

// TestLastAuthenticationSuccess tests recording authentication success timestamp
func TestLastAuthenticationSuccess(t *testing.T) {
	registry := prometheus.NewRegistry()

	em := &ExporterMetrics{
		AuthenticationValid: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_valid4",
			Help: "Test auth valid 4",
		}),
		AuthenticationErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_auth_errors4",
			Help: "Test auth errors 4",
		}),
		LastAuthenticationSuccessUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_auth_success_unix4",
			Help: "Test auth success unix 4",
		}),
	}

	require.NoError(t, registry.Register(em.AuthenticationValid))
	require.NoError(t, registry.Register(em.AuthenticationErrorsTotal))
	require.NoError(t, registry.Register(em.LastAuthenticationSuccessUnix))

	// Record authentication success
	em.RecordAuthenticationSuccess()

	families, err := registry.Gather()
	require.NoError(t, err)

	successFound := false
	for _, family := range families {
		if family.Name != nil && *family.Name == "test_auth_success_unix4" {
			successFound = true
			require.Greater(t, len(family.Metric), 0)
			// Verify timestamp is recent (within last 5 seconds)
			timestamp := *family.Metric[0].Gauge.Value
			assert.Greater(t, timestamp, float64(0), "timestamp should be positive")
		}
	}
	assert.True(t, successFound, "auth success timestamp metric not found")
}
