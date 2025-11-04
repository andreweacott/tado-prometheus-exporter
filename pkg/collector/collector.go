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
	tadoClient        *tado.ClientWithResponses
	metricDescriptors *metrics.MetricDescriptors
	scrapeTimeout     time.Duration
	homeID            string // Optional: filter to specific home
	log               *logger.Logger
	exporterMetrics   *metrics.ExporterMetrics // Optional: for internal health monitoring
}

// NewTadoCollector creates a new Tado metrics collector
func NewTadoCollector(
	tadoClient *tado.ClientWithResponses,
	metricDescriptors *metrics.MetricDescriptors,
	scrapeTimeout time.Duration,
	homeID string,
) *TadoCollector {
	return NewTadoCollectorWithLogger(tadoClient, metricDescriptors, scrapeTimeout, homeID, nil)
}

// NewTadoCollectorWithLogger creates a new Tado metrics collector with logging
func NewTadoCollectorWithLogger(
	tadoClient *tado.ClientWithResponses,
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
	meResponse, err := tc.tadoClient.GetMeWithResponse(ctx)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch user: %v", err)
		tc.log.Warn(errMsg)
		if tc.exporterMetrics != nil {
			tc.exporterMetrics.IncrementScrapeErrors()
		}
		// Return early if we can't even get the list of homes
		return fmt.Errorf("unable to retrieve user information: %w", err)
	}

	if meResponse.StatusCode() != 200 || meResponse.JSON200 == nil {
		errMsg := fmt.Sprintf("failed to fetch user: status code %d", meResponse.StatusCode())
		tc.log.Warn(errMsg)
		if tc.exporterMetrics != nil {
			tc.exporterMetrics.IncrementScrapeErrors()
		}
		return fmt.Errorf("failed to fetch user: status code %d", meResponse.StatusCode())
	}

	user := meResponse.JSON200
	if user.Homes == nil || len(*user.Homes) == 0 {
		tc.log.Warn("no homes found for user account")
		return fmt.Errorf("no homes found for user account")
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
	stateResponse, err := tc.tadoClient.GetHomeStateWithResponse(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get home state: %w", err)
	}

	if stateResponse.StatusCode() == 200 && stateResponse.JSON200 != nil {
		// Update resident presence metric
		// Presence is "HOME" or "AWAY"
		var presence float64
		if stateResponse.JSON200.Presence != nil && string(*stateResponse.JSON200.Presence) == "HOME" {
			presence = 1.0
		} else {
			presence = 0.0
		}
		tc.metricDescriptors.IsResidentPresent.Set(presence)
	}

	// Get weather (for solar intensity and outside temperature)
	weatherResponse, err := tc.tadoClient.GetWeatherWithResponse(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get weather: %w", err)
	}

	if weatherResponse.StatusCode() == 200 && weatherResponse.JSON200 != nil {
		weather := weatherResponse.JSON200

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
	var collectionErrors []string

	// Get all zones for this home
	zonesResponse, err := tc.tadoClient.GetZonesWithResponse(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get zones: %w", err)
	}

	if zonesResponse.StatusCode() != 200 || zonesResponse.JSON200 == nil {
		return fmt.Errorf("failed to get zones: status code %d", zonesResponse.StatusCode())
	}

	zones := *zonesResponse.JSON200

	// Get zone states
	statesResponse, err := tc.tadoClient.GetZoneStatesWithResponse(ctx, homeID)
	if err != nil {
		return fmt.Errorf("failed to get zone states: %w", err)
	}

	if statesResponse.StatusCode() != 200 || statesResponse.JSON200 == nil {
		return fmt.Errorf("no zone states available")
	}

	zoneStates := statesResponse.JSON200

	// Collect metrics for each zone - continue even if one fails
	zoneCount := 0
	zoneErrorCount := 0
	zoneStateMapRef := zoneStates.ZoneStates
	if zoneStateMapRef == nil {
		return fmt.Errorf("zone states map is nil")
	}
	zoneStateMapDeref := *zoneStateMapRef

	for _, zone := range zones {
		// Zone.Id is a pointer
		zoneID := zone.Id
		if zoneID == nil {
			tc.log.Warn("Zone ID is nil")
			continue
		}

		zoneIDStr := fmt.Sprintf("%d", *zoneID)
		zoneCount++

		// Get zone state from the map
		zoneState, ok := zoneStateMapDeref[zoneIDStr]
		if !ok {
			zoneErrorCount++
			tc.log.WithField("zone_id", zoneIDStr).Warn("No zone state found for zone")
			collectionErrors = append(collectionErrors, fmt.Sprintf("zone %s: state not found", zoneIDStr))
			continue
		}

		// Prepare label values (handle pointers)
		homeIDStr := fmt.Sprintf("%d", homeID)
		zoneName := zone.Name
		if zoneName == nil {
			zoneName = &[]string{"unknown"}[0]
		}
		zoneType := ""
		if zone.Type != nil {
			zoneType = string(*zone.Type)
		}

		// Collect measured temperature and humidity
		if zoneState.SensorDataPoints != nil {
			if zoneState.SensorDataPoints.InsideTemperature != nil {
				if zoneState.SensorDataPoints.InsideTemperature.Celsius != nil {
					tc.metricDescriptors.TemperatureMeasuredCelsius.WithLabelValues(
						homeIDStr, zoneIDStr, *zoneName, zoneType,
					).Set(float64(*zoneState.SensorDataPoints.InsideTemperature.Celsius))
				}

				if zoneState.SensorDataPoints.InsideTemperature.Fahrenheit != nil {
					tc.metricDescriptors.TemperatureMeasuredFahrenheit.WithLabelValues(
						homeIDStr, zoneIDStr, *zoneName, zoneType,
					).Set(float64(*zoneState.SensorDataPoints.InsideTemperature.Fahrenheit))
				}
			}

			if zoneState.SensorDataPoints.Humidity != nil && zoneState.SensorDataPoints.Humidity.Percentage != nil {
				tc.metricDescriptors.HumidityMeasuredPercentage.WithLabelValues(
					homeIDStr, zoneIDStr, *zoneName, zoneType,
				).Set(float64(*zoneState.SensorDataPoints.Humidity.Percentage))
			}
		}

		// Collect set temperature
		if zoneState.Setting != nil && zoneState.Setting.Temperature != nil {
			if zoneState.Setting.Temperature.Celsius != nil {
				tc.metricDescriptors.TemperatureSetCelsius.WithLabelValues(
					homeIDStr, zoneIDStr, *zoneName, zoneType,
				).Set(float64(*zoneState.Setting.Temperature.Celsius))
			}

			if zoneState.Setting.Temperature.Fahrenheit != nil {
				tc.metricDescriptors.TemperatureSetFahrenheit.WithLabelValues(
					homeIDStr, zoneIDStr, *zoneName, zoneType,
				).Set(float64(*zoneState.Setting.Temperature.Fahrenheit))
			}
		}

		// Collect heating power
		if zoneState.ActivityDataPoints != nil && zoneState.ActivityDataPoints.HeatingPower != nil &&
			zoneState.ActivityDataPoints.HeatingPower.Percentage != nil {
			tc.metricDescriptors.HeatingPowerPercentage.WithLabelValues(
				homeIDStr, zoneIDStr, *zoneName, zoneType,
			).Set(float64(*zoneState.ActivityDataPoints.HeatingPower.Percentage))
		}

		// Collect window status
		// OpenWindow is non-nil if a window is detected as open
		windowOpen := 0.0
		if zoneState.OpenWindow != nil {
			windowOpen = 1.0
		}
		tc.metricDescriptors.IsWindowOpen.WithLabelValues(
			homeIDStr, zoneIDStr, *zoneName, zoneType,
		).Set(windowOpen)

		// Collect zone powered status
		zonePowered := 0.0
		if zoneState.Setting != nil && zoneState.Setting.Power != nil && string(*zoneState.Setting.Power) == "ON" {
			zonePowered = 1.0
		}
		tc.metricDescriptors.IsZonePowered.WithLabelValues(
			homeIDStr, zoneIDStr, *zoneName, zoneType,
		).Set(zonePowered)
	}

	// Log zone collection summary
	if len(collectionErrors) > 0 {
		tc.log.Warn("Zone metrics collection completed with errors",
			"home_id", fmt.Sprintf("%d", homeID),
			"total_zones", zoneCount,
			"zones_with_errors", zoneErrorCount,
			"error_count", len(collectionErrors))
	}

	return nil
}
