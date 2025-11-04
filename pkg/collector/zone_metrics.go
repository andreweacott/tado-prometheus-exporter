// Package collector provides zone metric extraction helpers.
package collector

import (
	"fmt"

	"github.com/clambin/tado/v2"
)

// Validation constants for metric ranges
const (
	// Temperature (Celsius) - typical range for buildings
	MinValidTemperature float32 = -50
	MaxValidTemperature float32 = 60

	// Humidity (%) - always 0-100
	MinValidHumidity float32 = 0
	MaxValidHumidity float32 = 100

	// Power (%) - always 0-100
	MinValidPower float32 = 0
	MaxValidPower float32 = 100
)

// ZoneMetrics holds extracted metrics for a single zone
type ZoneMetrics struct {
	MeasuredTemperatureCelsius *float32
	MeasuredTemperatureFahrenheit *float32
	MeasuredHumidity *float32
	TargetTemperatureCelsius *float32
	TargetTemperatureFahrenheit *float32
	HeatingPowerPercentage *float32
	IsWindowOpen bool
	IsZonePowered bool
}

// extractZoneTemperature extracts the measured temperature from zone sensor data
func extractZoneTemperature(zoneState *tado.ZoneState) (*float32, *float32) {
	if zoneState == nil || zoneState.SensorDataPoints == nil {
		return nil, nil
	}
	if zoneState.SensorDataPoints.InsideTemperature == nil {
		return nil, nil
	}
	return zoneState.SensorDataPoints.InsideTemperature.Celsius,
		zoneState.SensorDataPoints.InsideTemperature.Fahrenheit
}

// extractZoneHumidity extracts the measured humidity from zone sensor data
func extractZoneHumidity(zoneState *tado.ZoneState) *float32 {
	if zoneState == nil || zoneState.SensorDataPoints == nil {
		return nil
	}
	if zoneState.SensorDataPoints.Humidity == nil {
		return nil
	}
	return zoneState.SensorDataPoints.Humidity.Percentage
}

// extractTargetTemperature extracts the target temperature from zone settings
func extractTargetTemperature(zoneState *tado.ZoneState) (*float32, *float32) {
	if zoneState == nil || zoneState.Setting == nil {
		return nil, nil
	}
	if zoneState.Setting.Temperature == nil {
		return nil, nil
	}
	return zoneState.Setting.Temperature.Celsius,
		zoneState.Setting.Temperature.Fahrenheit
}

// extractHeatingPower extracts the heating power percentage from activity data
func extractHeatingPower(zoneState *tado.ZoneState) *float32 {
	if zoneState == nil || zoneState.ActivityDataPoints == nil {
		return nil
	}
	if zoneState.ActivityDataPoints.HeatingPower == nil {
		return nil
	}
	return zoneState.ActivityDataPoints.HeatingPower.Percentage
}

// extractWindowOpenStatus determines if a window is open
func extractWindowOpenStatus(zoneState *tado.ZoneState) bool {
	if zoneState == nil {
		return false
	}
	return zoneState.OpenWindow != nil
}

// extractZonePowerStatus determines if a zone is powered on
func extractZonePowerStatus(zoneState *tado.ZoneState) bool {
	if zoneState == nil || zoneState.Setting == nil {
		return false
	}
	if zoneState.Setting.Power == nil {
		return false
	}
	return string(*zoneState.Setting.Power) == "ON"
}

// ExtractAllZoneMetrics extracts all metrics from a zone state
func ExtractAllZoneMetrics(zoneState *tado.ZoneState) *ZoneMetrics {
	tempC, tempF := extractZoneTemperature(zoneState)
	targetC, targetF := extractTargetTemperature(zoneState)

	return &ZoneMetrics{
		MeasuredTemperatureCelsius: tempC,
		MeasuredTemperatureFahrenheit: tempF,
		MeasuredHumidity: extractZoneHumidity(zoneState),
		TargetTemperatureCelsius: targetC,
		TargetTemperatureFahrenheit: targetF,
		HeatingPowerPercentage: extractHeatingPower(zoneState),
		IsWindowOpen: extractWindowOpenStatus(zoneState),
		IsZonePowered: extractZonePowerStatus(zoneState),
	}
}

// ValidationError represents a validation error for a metric
type ValidationError struct {
	Field  string
	Value  interface{}
	Reason string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s = %v, %s", ve.Field, ve.Value, ve.Reason)
}

// validateTemperature checks if a temperature is within valid bounds
func validateTemperature(temp float32, fieldName string) error {
	if temp < MinValidTemperature || temp > MaxValidTemperature {
		return &ValidationError{
			Field: fieldName,
			Value: temp,
			Reason: fmt.Sprintf("outside valid range [%g, %g]°C", MinValidTemperature, MaxValidTemperature),
		}
	}
	return nil
}

// validateHumidity checks if humidity is within valid bounds
func validateHumidity(humidity float32, fieldName string) error {
	if humidity < MinValidHumidity || humidity > MaxValidHumidity {
		return &ValidationError{
			Field: fieldName,
			Value: humidity,
			Reason: fmt.Sprintf("outside valid range [%g, %g]%%", MinValidHumidity, MaxValidHumidity),
		}
	}
	return nil
}

// validatePower checks if power percentage is within valid bounds
func validatePower(power float32, fieldName string) error {
	if power < MinValidPower || power > MaxValidPower {
		return &ValidationError{
			Field: fieldName,
			Value: power,
			Reason: fmt.Sprintf("outside valid range [%g, %g]%%", MinValidPower, MaxValidPower),
		}
	}
	return nil
}

// ValidateZoneMetrics validates extracted zone metrics
func ValidateZoneMetrics(metrics *ZoneMetrics) []error {
	var errors []error

	if metrics == nil {
		errors = append(errors, &ValidationError{
			Field: "metrics",
			Reason: "metrics object is nil",
		})
		return errors
	}

	// Validate measured temperature
	if metrics.MeasuredTemperatureCelsius != nil {
		if err := validateTemperature(*metrics.MeasuredTemperatureCelsius, "measured_temperature_celsius"); err != nil {
			errors = append(errors, err)
		}
	}

	// Validate measured humidity
	if metrics.MeasuredHumidity != nil {
		if err := validateHumidity(*metrics.MeasuredHumidity, "measured_humidity"); err != nil {
			errors = append(errors, err)
		}
	}

	// Validate target temperature
	if metrics.TargetTemperatureCelsius != nil {
		if err := validateTemperature(*metrics.TargetTemperatureCelsius, "target_temperature_celsius"); err != nil {
			errors = append(errors, err)
		}
	}

	// Validate heating power
	if metrics.HeatingPowerPercentage != nil {
		if err := validatePower(*metrics.HeatingPowerPercentage, "heating_power"); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
