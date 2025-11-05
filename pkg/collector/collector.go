// Package collector implements the Prometheus collector for Tado metrics.
//
// It provides:
//   - Prometheus collector interface implementation
//   - Tado API metrics fetching
//   - Graceful error handling with partial metric collection
//   - Exporter health metrics reporting
//
// The collector fetches metrics on-demand when Prometheus scrapes the /metrics
// endpoint. It continues collecting metrics even if some API calls fail, ensuring
// partial metrics are always available for monitoring and alerting.
package collector

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/logger"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/metrics"
	"github.com/clambin/tado/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// TadoCollector implements the prometheus.Collector interface
// It fetches Tado metrics on-demand when Prometheus scrapes the /metrics endpoint
type TadoCollector struct {
	tadoClient        TadoAPI
	metricDescriptors *metrics.MetricDescriptors
	scrapeTimeout     time.Duration
	homeID            string // Optional: filter to specific home
	log               *logger.Logger
	exporterMetrics   *metrics.ExporterMetrics // Optional: for internal health monitoring
}

// NewTadoCollector creates a new Tado metrics collector
func NewTadoCollector(
	tadoClient TadoAPI,
	metricDescriptors *metrics.MetricDescriptors,
	scrapeTimeout time.Duration,
	homeID string,
) *TadoCollector {
	return NewTadoCollectorWithLogger(tadoClient, metricDescriptors, scrapeTimeout, homeID, nil)
}

// NewTadoCollectorWithLogger creates a new Tado metrics collector with logging
func NewTadoCollectorWithLogger(
	tadoClient TadoAPI,
	metricDescriptors *metrics.MetricDescriptors,
	scrapeTimeout time.Duration,
	homeID string,
	log *logger.Logger,
) *TadoCollector {
	// Use noop logger if none provided
	if log == nil {
		noop, _ := logger.NewWithWriter("error", "text", io.Discard)
		log = noop
	}

	return &TadoCollector{
		tadoClient:        tadoClient,
		metricDescriptors: metricDescriptors,
		scrapeTimeout:     scrapeTimeout,
		homeID:            homeID,
		log:               log,
		exporterMetrics:   nil, // Will be set separately if needed
	}
}

// WithExporterMetrics adds exporter health metrics to the collector
func (tc *TadoCollector) WithExporterMetrics(em *metrics.ExporterMetrics) *TadoCollector {
	tc.exporterMetrics = em
	return tc
}

// Describe sends the super-set of all possible descriptors of metrics collected by this collector
func (tc *TadoCollector) Describe(ch chan<- *prometheus.Desc) {
	// Home-level metrics
	tc.metricDescriptors.IsResidentPresent.Describe(ch)
	tc.metricDescriptors.SolarIntensityPercentage.Describe(ch)
	tc.metricDescriptors.TemperatureOutsideCelsius.Describe(ch)
	tc.metricDescriptors.TemperatureOutsideFahrenheit.Describe(ch)

	// Zone-level metrics
	tc.metricDescriptors.TemperatureMeasuredCelsius.Describe(ch)
	tc.metricDescriptors.TemperatureMeasuredFahrenheit.Describe(ch)
	tc.metricDescriptors.HumidityMeasuredPercentage.Describe(ch)
	tc.metricDescriptors.TemperatureSetCelsius.Describe(ch)
	tc.metricDescriptors.TemperatureSetFahrenheit.Describe(ch)
	tc.metricDescriptors.HeatingPowerPercentage.Describe(ch)
	tc.metricDescriptors.IsWindowOpen.Describe(ch)
	tc.metricDescriptors.IsZonePowered.Describe(ch)

	// Exporter health metrics if configured
	if tc.exporterMetrics != nil {
		tc.exporterMetrics.ScrapeDurationSeconds.Describe(ch)
		tc.exporterMetrics.ScrapeErrorsTotal.Describe(ch)
		tc.exporterMetrics.BuildInfo.Describe(ch)
		tc.exporterMetrics.AuthenticationValid.Describe(ch)
		tc.exporterMetrics.AuthenticationErrorsTotal.Describe(ch)
		tc.exporterMetrics.LastAuthenticationSuccessUnix.Describe(ch)
	}
}

// Collect is called by the Prometheus client when scraping /metrics
// It fetches current metrics from Tado API and sends them to the channel
func (tc *TadoCollector) Collect(ch chan<- prometheus.Metric) {
	// Create context with timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(context.Background(), tc.scrapeTimeout)
	defer cancel()

	// Record scrape duration if exporter metrics are configured
	var startTime time.Time
	if tc.exporterMetrics != nil {
		startTime = time.Now()
	}

	// Fetch metrics from Tado API
	if err := tc.fetchAndCollectMetrics(ctx); err != nil {
		tc.log.Warn("Failed to collect Tado metrics", "error", err.Error())
		// Increment error counter if exporter metrics are configured
		if tc.exporterMetrics != nil {
			tc.exporterMetrics.IncrementScrapeErrors()
		}
		// Don't return - Prometheus will use last known values
	}

	// Record scrape duration if exporter metrics are configured
	if tc.exporterMetrics != nil {
		duration := time.Since(startTime).Seconds()
		tc.exporterMetrics.RecordScrapeDuration(duration)
	}

	// Send collected metrics to channel
	// Home-level metrics
	tc.metricDescriptors.IsResidentPresent.Collect(ch)
	tc.metricDescriptors.SolarIntensityPercentage.Collect(ch)
	tc.metricDescriptors.TemperatureOutsideCelsius.Collect(ch)
	tc.metricDescriptors.TemperatureOutsideFahrenheit.Collect(ch)

	// Zone-level metrics
	tc.metricDescriptors.TemperatureMeasuredCelsius.Collect(ch)
	tc.metricDescriptors.TemperatureMeasuredFahrenheit.Collect(ch)
	tc.metricDescriptors.HumidityMeasuredPercentage.Collect(ch)
	tc.metricDescriptors.TemperatureSetCelsius.Collect(ch)
	tc.metricDescriptors.TemperatureSetFahrenheit.Collect(ch)
	tc.metricDescriptors.HeatingPowerPercentage.Collect(ch)
	tc.metricDescriptors.IsWindowOpen.Collect(ch)
	tc.metricDescriptors.IsZonePowered.Collect(ch)

	// Send exporter health metrics to channel if configured
	if tc.exporterMetrics != nil {
		tc.exporterMetrics.ScrapeDurationSeconds.Collect(ch)
		tc.exporterMetrics.ScrapeErrorsTotal.Collect(ch)
		tc.exporterMetrics.BuildInfo.Collect(ch)
		tc.exporterMetrics.AuthenticationValid.Collect(ch)
		tc.exporterMetrics.AuthenticationErrorsTotal.Collect(ch)
		tc.exporterMetrics.LastAuthenticationSuccessUnix.Collect(ch)
	}
}

// fetchAndCollectMetrics fetches metrics from Tado API and updates metric values
// This function continues collecting metrics even when individual API calls fail,
// ensuring partial metrics are always available for alerting and monitoring.
func (tc *TadoCollector) fetchAndCollectMetrics(ctx context.Context) error {
	var collectionErrors []string

	// Get current user and homes
	user, err := tc.tadoClient.GetMe(ctx)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch user: %v", err)
		tc.log.Warn(errMsg)
		if tc.exporterMetrics != nil {
			tc.exporterMetrics.IncrementScrapeErrors()
			tc.exporterMetrics.IncrementAuthenticationErrors()
			tc.exporterMetrics.SetAuthenticationValid(false)
		}
		// Return early if we can't even get the list of homes
		return fmt.Errorf("unable to retrieve user information: %w", err)
	}
	if user.Homes == nil || len(*user.Homes) == 0 {
		tc.log.Warn("no homes found for user account")
		if tc.exporterMetrics != nil {
			tc.exporterMetrics.IncrementAuthenticationErrors()
			tc.exporterMetrics.SetAuthenticationValid(false)
		}
		return fmt.Errorf("no homes found for user account")
	}

	// Authentication succeeded - set metric to valid and record success timestamp
	if tc.exporterMetrics != nil {
		tc.exporterMetrics.SetAuthenticationValid(true)
		tc.exporterMetrics.RecordAuthenticationSuccess()
	}

	// Collect metrics from each home - continue even if one fails
	homeCount := 0
	homeErrorCount := 0
	for _, userHome := range *user.Homes {
		// Get home ID value (might be pointer)
		homeID := userHome.Id
		if homeID == nil {
			continue
		}

		// Filter to specific home if specified
		if tc.homeID != "" && fmt.Sprintf("%d", *homeID) != tc.homeID {
			continue
		}

		homeCount++
		homeIDStr := fmt.Sprintf("%d", *homeID)

		// Collect home-level metrics - continue if fails
		if err := tc.collectHomeMetrics(ctx, *homeID); err != nil {
			homeErrorCount++
			errMsg := fmt.Sprintf("home metrics for %s: %v", homeIDStr, err)
			tc.log.WithField("home_id", homeIDStr).Warn("Failed to collect home metrics", "error", err.Error())
			collectionErrors = append(collectionErrors, errMsg)
			// Continue to collect zone metrics even if home metrics fail
		}

		// Collect zone-level metrics - continue if fails
		if err := tc.collectZoneMetrics(ctx, *homeID); err != nil {
			errMsg := fmt.Sprintf("zone metrics for %s: %v", homeIDStr, err)
			tc.log.WithField("home_id", homeIDStr).Warn("Failed to collect zone metrics", "error", err.Error())
			collectionErrors = append(collectionErrors, errMsg)
			// Continue even if zone metrics fail
		}
	}

	// If we collected from at least some homes, consider it a partial success
	// Log warnings about failures but don't treat as a complete failure
	if len(collectionErrors) > 0 {
		tc.log.Warn("Scrape completed with errors",
			"total_homes", homeCount,
			"homes_with_errors", homeErrorCount,
			"error_count", len(collectionErrors))
	}

	return nil
}

// collectHomeMetrics collects home-level metrics (presence, weather)
func (tc *TadoCollector) collectHomeMetrics(ctx context.Context, homeID tado.HomeId) error {
	// Get home state (for resident presence)
	homeState, err := tc.tadoClient.GetHomeState(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get home state: %w", err)
	}

	if homeState != nil {
		// Update resident presence metric
		// Presence is "HOME" or "AWAY"
		var presence float64
		if homeState.Presence != nil && string(*homeState.Presence) == "HOME" {
			presence = 1.0
		} else {
			presence = 0.0
		}
		tc.metricDescriptors.IsResidentPresent.Set(presence)
	}

	// Get weather (for solar intensity and outside temperature)
	weather, err := tc.tadoClient.GetWeather(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get weather: %w", err)
	}

	if weather != nil {

		// Update solar intensity metric
		if weather.SolarIntensity != nil && weather.SolarIntensity.Percentage != nil {
			tc.metricDescriptors.SolarIntensityPercentage.Set(float64(*weather.SolarIntensity.Percentage))
		}

		// Update outside temperature metrics
		if weather.OutsideTemperature != nil {
			if weather.OutsideTemperature.Celsius != nil {
				tc.metricDescriptors.TemperatureOutsideCelsius.Set(float64(*weather.OutsideTemperature.Celsius))
			}
			if weather.OutsideTemperature.Fahrenheit != nil {
				tc.metricDescriptors.TemperatureOutsideFahrenheit.Set(float64(*weather.OutsideTemperature.Fahrenheit))
			}
		}
	}

	return nil
}

// collectZoneMetrics collects zone-level metrics (temperature, humidity, heating power, window status)
// This function continues collecting metrics for each zone even if one zone fails,
// ensuring partial metrics are available even if some zones have errors.
func (tc *TadoCollector) collectZoneMetrics(ctx context.Context, homeID tado.HomeId) error {
	// Get all zones for this home
	zones, err := tc.tadoClient.GetZones(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get zones: %w", err)
	}

	// Get zone states
	zoneStates, err := tc.tadoClient.GetZoneStates(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get zone states: %w", err)
	}

	if zoneStates == nil || zoneStates.ZoneStates == nil {
		return fmt.Errorf("zone states are nil")
	}

	// Collect metrics for each zone - continue even if one fails
	homeIDStr := fmt.Sprintf("%d", homeID)
	zoneCount := 0
	zoneErrorCount := 0

	for _, zone := range zones {
		if err := tc.collectSingleZoneMetrics(homeIDStr, zone, *zoneStates.ZoneStates); err != nil {
			zoneErrorCount++
			tc.log.WithField("zone_id", fmt.Sprintf("%d", *zone.Id)).Warn("Failed to collect zone metrics", "error", err.Error())
		}
		zoneCount++
	}

	// Log zone collection summary
	if zoneErrorCount > 0 {
		tc.log.Warn("Zone metrics collection completed with errors",
			"home_id", homeIDStr,
			"total_zones", zoneCount,
			"zones_with_errors", zoneErrorCount)
	}

	return nil
}

// collectSingleZoneMetrics collects metrics for a single zone
func (tc *TadoCollector) collectSingleZoneMetrics(homeIDStr string, zone tado.Zone, zoneStatesMap map[string]tado.ZoneState) error {
	// Validate zone ID
	if zone.Id == nil {
		return fmt.Errorf("zone ID is nil")
	}

	zoneIDStr := fmt.Sprintf("%d", *zone.Id)

	// Get zone state from the map
	zoneState, ok := zoneStatesMap[zoneIDStr]
	if !ok {
		return fmt.Errorf("zone state not found in map")
	}

	// Extract zone metadata for labels
	zoneName := zone.Name
	if zoneName == nil {
		zoneName = &[]string{"unknown"}[0]
	}
	zoneType := ""
	if zone.Type != nil {
		zoneType = string(*zone.Type)
	}

	// Extract all metrics from zone state
	metrics := ExtractAllZoneMetrics(&zoneState)

	// Validate extracted metrics
	validationErrors := ValidateZoneMetrics(metrics)
	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			tc.log.WithField("zone_id", zoneIDStr).Warn("Zone metric validation failed", "error", err.Error())
		}
	}

	// Record all metrics
	labels := []string{homeIDStr, zoneIDStr, *zoneName, zoneType}
	tc.recordMeasuredTemperatureMetrics(zoneIDStr, labels, metrics)
	tc.recordMeasuredHumidityMetric(zoneIDStr, labels, metrics)
	tc.recordTargetTemperatureMetrics(zoneIDStr, labels, metrics)
	tc.recordHeatingPowerMetric(zoneIDStr, labels, metrics)
	tc.recordWindowStatusMetric(labels, metrics)
	tc.recordZonePoweredStatusMetric(labels, metrics)

	return nil
}

// recordMeasuredTemperatureMetrics records both Celsius and Fahrenheit measured temperatures
func (tc *TadoCollector) recordMeasuredTemperatureMetrics(zoneIDStr string, labels []string, metrics *ZoneMetrics) {
	if metrics.MeasuredTemperatureCelsius != nil {
		if err := validateTemperature(*metrics.MeasuredTemperatureCelsius, "measured_temperature_celsius"); err != nil {
			tc.log.WithField("zone_id", zoneIDStr).Warn("Invalid measured temperature, skipping metric", "value", *metrics.MeasuredTemperatureCelsius, "error", err.Error())
		} else {
			tc.metricDescriptors.TemperatureMeasuredCelsius.WithLabelValues(labels...).Set(float64(*metrics.MeasuredTemperatureCelsius))
		}
	}

	if metrics.MeasuredTemperatureFahrenheit != nil {
		tc.metricDescriptors.TemperatureMeasuredFahrenheit.WithLabelValues(labels...).Set(float64(*metrics.MeasuredTemperatureFahrenheit))
	}
}

// recordMeasuredHumidityMetric records the measured humidity
func (tc *TadoCollector) recordMeasuredHumidityMetric(zoneIDStr string, labels []string, metrics *ZoneMetrics) {
	if metrics.MeasuredHumidity != nil {
		if err := validateHumidity(*metrics.MeasuredHumidity, "measured_humidity"); err != nil {
			tc.log.WithField("zone_id", zoneIDStr).Warn("Invalid measured humidity, skipping metric", "value", *metrics.MeasuredHumidity, "error", err.Error())
		} else {
			tc.metricDescriptors.HumidityMeasuredPercentage.WithLabelValues(labels...).Set(float64(*metrics.MeasuredHumidity))
		}
	}
}

// recordTargetTemperatureMetrics records both Celsius and Fahrenheit target temperatures
func (tc *TadoCollector) recordTargetTemperatureMetrics(zoneIDStr string, labels []string, metrics *ZoneMetrics) {
	if metrics.TargetTemperatureCelsius != nil {
		if err := validateTemperature(*metrics.TargetTemperatureCelsius, "target_temperature_celsius"); err != nil {
			tc.log.WithField("zone_id", zoneIDStr).Warn("Invalid target temperature, skipping metric", "value", *metrics.TargetTemperatureCelsius, "error", err.Error())
		} else {
			tc.metricDescriptors.TemperatureSetCelsius.WithLabelValues(labels...).Set(float64(*metrics.TargetTemperatureCelsius))
		}
	}

	if metrics.TargetTemperatureFahrenheit != nil {
		tc.metricDescriptors.TemperatureSetFahrenheit.WithLabelValues(labels...).Set(float64(*metrics.TargetTemperatureFahrenheit))
	}
}

// recordHeatingPowerMetric records the heating power percentage
func (tc *TadoCollector) recordHeatingPowerMetric(zoneIDStr string, labels []string, metrics *ZoneMetrics) {
	if metrics.HeatingPowerPercentage != nil {
		if err := validatePower(*metrics.HeatingPowerPercentage, "heating_power"); err != nil {
			tc.log.WithField("zone_id", zoneIDStr).Warn("Invalid heating power, skipping metric", "value", *metrics.HeatingPowerPercentage, "error", err.Error())
		} else {
			tc.metricDescriptors.HeatingPowerPercentage.WithLabelValues(labels...).Set(float64(*metrics.HeatingPowerPercentage))
		}
	}
}

// recordWindowStatusMetric records whether the window is open (1) or closed (0)
func (tc *TadoCollector) recordWindowStatusMetric(labels []string, metrics *ZoneMetrics) {
	windowOpen := 0.0
	if metrics.IsWindowOpen {
		windowOpen = 1.0
	}
	tc.metricDescriptors.IsWindowOpen.WithLabelValues(labels...).Set(windowOpen)
}

// recordZonePoweredStatusMetric records whether the zone is powered (1) or off (0)
func (tc *TadoCollector) recordZonePoweredStatusMetric(labels []string, metrics *ZoneMetrics) {
	zonePowered := 0.0
	if metrics.IsZonePowered {
		zonePowered = 1.0
	}
	tc.metricDescriptors.IsZonePowered.WithLabelValues(labels...).Set(zonePowered)
}
