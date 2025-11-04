// Package metrics defines Prometheus metrics for the exporter.
//
// It provides:
//   - Tado API metrics (temperature, humidity, heating power, etc.)
//   - Exporter health metrics (scrape duration, errors, authentication status)
//   - Metric registration with Prometheus
//
// The package creates metrics for:
//   - Home-level data: resident presence, weather (solar intensity, outside temperature)
//   - Zone-level data: measured/set temperature, humidity, heating power, window/power status
//   - Exporter health: collection performance, error tracking, authentication status
//
// Example usage:
//
//	metricDescs, err := metrics.NewMetricDescriptors()
//	if err != nil {
//		log.Fatal(err)
//	}
//	exporterMetrics, err := metrics.NewExporterMetrics()
//	if err != nil {
//		log.Fatal(err)
//	}
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricDescriptors holds all Prometheus metric descriptors for Tado
type MetricDescriptors struct {
	// Home-level metrics
	IsResidentPresent            prometheus.Gauge
	SolarIntensityPercentage     prometheus.Gauge
	TemperatureOutsideCelsius    prometheus.Gauge
	TemperatureOutsideFahrenheit prometheus.Gauge

	// Zone-level metrics (with labels: zone_id, zone_name, zone_type)
	TemperatureMeasuredCelsius    prometheus.GaugeVec
	TemperatureMeasuredFahrenheit prometheus.GaugeVec
	HumidityMeasuredPercentage    prometheus.GaugeVec
	TemperatureSetCelsius         prometheus.GaugeVec
	TemperatureSetFahrenheit      prometheus.GaugeVec
	HeatingPowerPercentage        prometheus.GaugeVec
	IsWindowOpen                  prometheus.GaugeVec
	IsZonePowered                 prometheus.GaugeVec
}

// NewMetricDescriptors creates and registers all Prometheus metrics
func NewMetricDescriptors() (*MetricDescriptors, error) {
	md := &MetricDescriptors{
		// Home-level metrics (no labels)
		IsResidentPresent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_is_resident_present",
			Help: "Whether anyone is home (1 = home, 0 = away)",
		}),

		SolarIntensityPercentage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_solar_intensity_percentage",
			Help: "Solar radiation intensity as a percentage (0-100%)",
		}),

		TemperatureOutsideCelsius: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_temperature_outside_celsius",
			Help: "Outside temperature in Celsius",
		}),

		TemperatureOutsideFahrenheit: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_temperature_outside_fahrenheit",
			Help: "Outside temperature in Fahrenheit",
		}),

		// Zone-level metrics (with labels: zone_id, zone_name, zone_type)
		TemperatureMeasuredCelsius: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_measured_celsius",
				Help: "Measured temperature in Celsius",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		TemperatureMeasuredFahrenheit: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_measured_fahrenheit",
				Help: "Measured temperature in Fahrenheit",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		HumidityMeasuredPercentage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_humidity_measured_percentage",
				Help: "Measured relative humidity as a percentage (0-100%)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		TemperatureSetCelsius: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_set_celsius",
				Help: "Set/target temperature in Celsius",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		TemperatureSetFahrenheit: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_set_fahrenheit",
				Help: "Set/target temperature in Fahrenheit",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		HeatingPowerPercentage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_heating_power_percentage",
				Help: "Heating power as a percentage (0-100%)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		IsWindowOpen: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_is_window_open",
				Help: "Whether the window is open (1 = open, 0 = closed)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		IsZonePowered: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_is_zone_powered",
				Help: "Whether the zone is powered (1 = on, 0 = off)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),
	}

	// Register all metrics with Prometheus default registry
	if err := md.Register(); err != nil {
		return nil, err
	}

	return md, nil
}

// NewMetricDescriptorsUnregistered creates metric descriptors without registering them
// This is useful for testing where each test needs isolated registries
func NewMetricDescriptorsUnregistered() (*MetricDescriptors, error) {
	md := &MetricDescriptors{
		// Home-level metrics (no labels)
		IsResidentPresent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_is_resident_present",
			Help: "Whether anyone is home (1 = home, 0 = away)",
		}),

		SolarIntensityPercentage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_solar_intensity_percentage",
			Help: "Solar radiation intensity as a percentage (0-100%)",
		}),

		TemperatureOutsideCelsius: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_temperature_outside_celsius",
			Help: "Outside temperature in Celsius",
		}),

		TemperatureOutsideFahrenheit: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "tado_temperature_outside_fahrenheit",
			Help: "Outside temperature in Fahrenheit",
		}),

		// Zone-level metrics (with labels: zone_id, zone_name, zone_type)
		TemperatureMeasuredCelsius: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_measured_celsius",
				Help: "Measured temperature in Celsius",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		TemperatureMeasuredFahrenheit: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_measured_fahrenheit",
				Help: "Measured temperature in Fahrenheit",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		HumidityMeasuredPercentage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_humidity_measured_percentage",
				Help: "Measured relative humidity as a percentage (0-100%)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		TemperatureSetCelsius: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_set_celsius",
				Help: "Set/target temperature in Celsius",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		TemperatureSetFahrenheit: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_temperature_set_fahrenheit",
				Help: "Set/target temperature in Fahrenheit",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		HeatingPowerPercentage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_heating_power_percentage",
				Help: "Heating power as a percentage (0-100%)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		IsWindowOpen: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_is_window_open",
				Help: "Whether the window is open (1 = open, 0 = closed)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),

		IsZonePowered: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tado_is_zone_powered",
				Help: "Whether the zone is powered (1 = on, 0 = off)",
			},
			[]string{"home_id", "zone_id", "zone_name", "zone_type"},
		),
	}

	// Note: We do NOT register here - caller must use RegisterWith()
	return md, nil
}

// RegisterWith registers all metrics with the provided Prometheus registry
func (md *MetricDescriptors) RegisterWith(registerer prometheus.Registerer) error {
	// Home-level metrics
	if err := registerer.Register(md.IsResidentPresent); err != nil {
		return err
	}
	if err := registerer.Register(md.SolarIntensityPercentage); err != nil {
		return err
	}
	if err := registerer.Register(md.TemperatureOutsideCelsius); err != nil {
		return err
	}
	if err := registerer.Register(md.TemperatureOutsideFahrenheit); err != nil {
		return err
	}

	// Zone-level metrics
	if err := registerer.Register(&md.TemperatureMeasuredCelsius); err != nil {
		return err
	}
	if err := registerer.Register(&md.TemperatureMeasuredFahrenheit); err != nil {
		return err
	}
	if err := registerer.Register(&md.HumidityMeasuredPercentage); err != nil {
		return err
	}
	if err := registerer.Register(&md.TemperatureSetCelsius); err != nil {
		return err
	}
	if err := registerer.Register(&md.TemperatureSetFahrenheit); err != nil {
		return err
	}
	if err := registerer.Register(&md.HeatingPowerPercentage); err != nil {
		return err
	}
	if err := registerer.Register(&md.IsWindowOpen); err != nil {
		return err
	}
	if err := registerer.Register(&md.IsZonePowered); err != nil {
		return err
	}

	return nil
}

// Register registers all metrics with the Prometheus default registry
// Deprecated: Use RegisterWith instead for custom registries
func (md *MetricDescriptors) Register() error {
	return md.RegisterWith(prometheus.DefaultRegisterer)
}

// Reset clears all metric values (useful for testing)
func (md *MetricDescriptors) Reset() {
	md.IsResidentPresent.Set(0)
	md.SolarIntensityPercentage.Set(0)
	md.TemperatureOutsideCelsius.Set(0)
	md.TemperatureOutsideFahrenheit.Set(0)

	md.TemperatureMeasuredCelsius.Reset()
	md.TemperatureMeasuredFahrenheit.Reset()
	md.HumidityMeasuredPercentage.Reset()
	md.TemperatureSetCelsius.Reset()
	md.TemperatureSetFahrenheit.Reset()
	md.HeatingPowerPercentage.Reset()
	md.IsWindowOpen.Reset()
	md.IsZonePowered.Reset()
}

// CelsiusToFahrenheit converts Celsius to Fahrenheit
func CelsiusToFahrenheit(celsius float64) float64 {
	return celsius*9/5 + 32
}
