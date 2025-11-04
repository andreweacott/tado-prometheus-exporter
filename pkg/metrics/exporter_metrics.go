// Package metrics defines Prometheus metrics for the Tado exporter
//
// IMPORTANT: Metric Update Pattern
// All methods in ExporterMetrics must be explicitly called by the collector
// to ensure metrics accurately reflect the exporter's state.
// See pkg/collector/collector.go fetchAndCollectMetrics() for implementation.
//
// Metric Methods and Where They're Called:
// 1. RecordScrapeDuration(duration) - in Collect() after metrics fetch
// 2. IncrementScrapeErrors() - when GetMe fails or collection errors occur
// 3. SetAuthenticationValid(valid) - on GetMe success (true) or failure (false)
// 4. IncrementAuthenticationErrors() - when GetMe fails or no homes found
// 5. RecordAuthenticationSuccess() - when GetMe succeeds with homes
//
// If adding new metrics, ensure they're called in the appropriate places
// in collector.go and covered by tests.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ExporterMetrics holds Prometheus metrics for exporter internal monitoring
type ExporterMetrics struct {
	// Scrape duration histogram (in seconds)
	ScrapeDurationSeconds prometheus.Histogram

	// Scrape error counter
	ScrapeErrorsTotal prometheus.Counter

	// Build info gauge
	BuildInfo prometheus.Gauge

	// Authentication status gauge (1 = valid, 0 = invalid/expired)
	AuthenticationValid prometheus.Gauge

	// Authentication error counter
	AuthenticationErrorsTotal prometheus.Counter

	// Last successful authentication timestamp (unix seconds)
	LastAuthenticationSuccessUnix prometheus.Gauge
}

// NewExporterMetrics creates and registers exporter health metrics
func NewExporterMetrics() (*ExporterMetrics, error) {
	em := &ExporterMetrics{
		// Scrape duration histogram with buckets: 100ms, 500ms, 1s, 2s, 5s, 10s
		ScrapeDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "tado_exporter_scrape_duration_seconds",
			Help:    "Time taken to collect metrics from Tado API in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 6), // 0.1, 0.2, 0.4, 0.8, 1.6, 3.2
		}),

		// Scrape error counter
		ScrapeErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tado_exporter_scrape_errors_total",
			Help: "Total number of errors while collecting metrics from Tado API",
		}),

		// Build info gauge
		BuildInfo: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_exporter_build_info",
			Help: "Build information for the exporter (value is always 1)",
		}),

		// Authentication status gauge (1 = valid, 0 = invalid/expired)
		AuthenticationValid: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_exporter_authentication_valid",
			Help: "Set to 1 if Tado authentication is valid and metrics are being collected, 0 if authentication failed or no homes found",
		}),

		// Authentication error counter
		AuthenticationErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tado_exporter_authentication_errors_total",
			Help: "Total number of authentication failures or token refresh attempts",
		}),

		// Last successful authentication timestamp
		LastAuthenticationSuccessUnix: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_exporter_last_authentication_success_unix",
			Help: "Unix timestamp of the last successful authentication",
		}),
	}

	// Register metrics
	if err := em.Register(); err != nil {
		return nil, err
	}

	// Set build info to 1
	em.BuildInfo.Set(1)

	// Initialize authentication status to invalid (will be set to 1 once authentication succeeds during first scrape)
	em.AuthenticationValid.Set(0)

	return em, nil
}

// Register registers exporter metrics with Prometheus
func (em *ExporterMetrics) Register() error {
	if err := prometheus.DefaultRegisterer.Register(em.ScrapeDurationSeconds); err != nil {
		return err
	}
	if err := prometheus.DefaultRegisterer.Register(em.ScrapeErrorsTotal); err != nil {
		return err
	}
	if err := prometheus.DefaultRegisterer.Register(em.BuildInfo); err != nil {
		return err
	}
	if err := prometheus.DefaultRegisterer.Register(em.AuthenticationValid); err != nil {
		return err
	}
	if err := prometheus.DefaultRegisterer.Register(em.AuthenticationErrorsTotal); err != nil {
		return err
	}
	if err := prometheus.DefaultRegisterer.Register(em.LastAuthenticationSuccessUnix); err != nil {
		return err
	}
	return nil
}

// RecordScrapeDuration records the duration of a metrics collection attempt
func (em *ExporterMetrics) RecordScrapeDuration(duration float64) {
	em.ScrapeDurationSeconds.Observe(duration)
}

// IncrementScrapeErrors increments the error counter
func (em *ExporterMetrics) IncrementScrapeErrors() {
	em.ScrapeErrorsTotal.Inc()
}

// SetAuthenticationValid sets the authentication status gauge
func (em *ExporterMetrics) SetAuthenticationValid(valid bool) {
	if valid {
		em.AuthenticationValid.Set(1)
	} else {
		em.AuthenticationValid.Set(0)
	}
}

// IncrementAuthenticationErrors increments the authentication error counter
func (em *ExporterMetrics) IncrementAuthenticationErrors() {
	em.AuthenticationErrorsTotal.Inc()
}

// RecordAuthenticationSuccess records a successful authentication by setting the timestamp
func (em *ExporterMetrics) RecordAuthenticationSuccess() {
	em.LastAuthenticationSuccessUnix.Set(float64(time.Now().Unix()))
}
