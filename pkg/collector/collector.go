package collector

import (
	"context"
	"fmt"
	"time"

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
}

// NewTadoCollector creates a new Tado metrics collector
func NewTadoCollector(
	tadoClient *tado.ClientWithResponses,
	metricDescriptors *metrics.MetricDescriptors,
	scrapeTimeout time.Duration,
	homeID string,
) *TadoCollector {
	return &TadoCollector{
		tadoClient:        tadoClient,
		metricDescriptors: metricDescriptors,
		scrapeTimeout:     scrapeTimeout,
		homeID:            homeID,
	}
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
}

// Collect is called by the Prometheus client when scraping /metrics
// It fetches current metrics from Tado API and sends them to the channel
func (tc *TadoCollector) Collect(ch chan<- prometheus.Metric) {
	// Create context with timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(context.Background(), tc.scrapeTimeout)
	defer cancel()

	// Fetch metrics from Tado API
	if err := tc.fetchAndCollectMetrics(ctx); err != nil {
		fmt.Printf("Warning: failed to collect Tado metrics: %v\n", err)
		// Don't return - Prometheus will use last known values
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
}

// fetchAndCollectMetrics fetches metrics from Tado API and updates metric values
func (tc *TadoCollector) fetchAndCollectMetrics(ctx context.Context) error {
	// Get current user and homes
	meResponse, err := tc.tadoClient.GetMeWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	if meResponse.StatusCode() != 200 || meResponse.JSON200 == nil {
		return fmt.Errorf("failed to fetch user: status code %d", meResponse.StatusCode())
	}

	user := meResponse.JSON200
	if user.Homes == nil || len(*user.Homes) == 0 {
		return fmt.Errorf("no homes found for user account")
	}

	// Collect metrics from each home
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

		// Collect home-level metrics
		if err := tc.collectHomeMetrics(ctx, *homeID); err != nil {
			fmt.Printf("Warning: failed to collect home metrics for home %d: %v\n", *homeID, err)
			// Continue collecting zone metrics even if home metrics fail
		}

		// Collect zone-level metrics
		if err := tc.collectZoneMetrics(ctx, *homeID); err != nil {
			fmt.Printf("Warning: failed to collect zone metrics for home %d: %v\n", *homeID, err)
			// Continue even if zone metrics fail
		}
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
func (tc *TadoCollector) collectZoneMetrics(ctx context.Context, homeID tado.HomeId) error {
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

	// Collect metrics for each zone
	for _, zone := range zones {
		// Get zone state from the map (dereference if pointer)
		zoneStateMap := zoneStates.ZoneStates
		if zoneStateMap == nil {
			fmt.Printf("Warning: zone states map is nil\n")
			return fmt.Errorf("zone states map is nil")
		}

		zoneStateMapDeref := *zoneStateMap

		// Zone.Id is a pointer
		zoneID := zone.Id
		if zoneID == nil {
			fmt.Printf("Warning: zone ID is nil\n")
			continue
		}

		zoneIDStr := fmt.Sprintf("%d", *zoneID)
		zoneState, ok := zoneStateMapDeref[zoneIDStr]
		if !ok {
			fmt.Printf("Warning: no zone state found for zone %s\n", zoneIDStr)
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

	return nil
}
