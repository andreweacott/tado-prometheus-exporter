package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateTemperature tests temperature validation
func TestValidateTemperature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		temp    float32
		wantErr bool
	}{
		{
			name:    "valid temp low end",
			temp:    -50,
			wantErr: false,
		},
		{
			name:    "valid temp middle",
			temp:    20.5,
			wantErr: false,
		},
		{
			name:    "valid temp high end",
			temp:    60,
			wantErr: false,
		},
		{
			name:    "too cold",
			temp:    -51,
			wantErr: true,
		},
		{
			name:    "too hot",
			temp:    61,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemperature(tt.temp, "test_temp")
			if tt.wantErr {
				assert.Error(t, err)
				assert.IsType(t, &ValidationError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateHumidity tests humidity validation
func TestValidateHumidity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		humidity float32
		wantErr  bool
	}{
		{
			name:     "valid humidity low",
			humidity: 0,
			wantErr:  false,
		},
		{
			name:     "valid humidity middle",
			humidity: 45.5,
			wantErr:  false,
		},
		{
			name:     "valid humidity high",
			humidity: 100,
			wantErr:  false,
		},
		{
			name:     "negative humidity",
			humidity: -1,
			wantErr:  true,
		},
		{
			name:     "humidity over 100",
			humidity: 101,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHumidity(tt.humidity, "test_humidity")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePower tests power validation
func TestValidatePower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		power   float32
		wantErr bool
	}{
		{
			name:    "valid power low",
			power:   0,
			wantErr: false,
		},
		{
			name:    "valid power middle",
			power:   50,
			wantErr: false,
		},
		{
			name:    "valid power high",
			power:   100,
			wantErr: false,
		},
		{
			name:    "negative power",
			power:   -1,
			wantErr: true,
		},
		{
			name:    "power over 100",
			power:   101,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePower(tt.power, "test_power")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateZoneMetricsNil tests metrics validation with nil metrics
func TestValidateZoneMetricsNil(t *testing.T) {
	t.Parallel()

	errors := ValidateZoneMetrics(nil)
	assert.Equal(t, 1, len(errors))
	assert.Error(t, errors[0])
}

// TestValidateZoneMetricsValid tests metrics validation with valid metrics
func TestValidateZoneMetricsValid(t *testing.T) {
	t.Parallel()

	temp := float32(20.5)
	humidity := float32(45.0)
	power := float32(50.0)

	metrics := &ZoneMetrics{
		MeasuredTemperatureCelsius: &temp,
		MeasuredHumidity:           &humidity,
		HeatingPowerPercentage:     &power,
	}

	errors := ValidateZoneMetrics(metrics)
	assert.Equal(t, 0, len(errors))
}

// TestValidateZoneMetricsInvalidTemperature tests metrics validation with invalid temperature
func TestValidateZoneMetricsInvalidTemperature(t *testing.T) {
	t.Parallel()

	badTemp := float32(100.0) // Out of range
	metrics := &ZoneMetrics{
		MeasuredTemperatureCelsius: &badTemp,
	}

	errors := ValidateZoneMetrics(metrics)
	assert.Equal(t, 1, len(errors))
	assert.Error(t, errors[0])
}

// TestValidateZoneMetricsInvalidHumidity tests metrics validation with invalid humidity
func TestValidateZoneMetricsInvalidHumidity(t *testing.T) {
	t.Parallel()

	badHumidity := float32(150.0) // Out of range
	metrics := &ZoneMetrics{
		MeasuredHumidity: &badHumidity,
	}

	errors := ValidateZoneMetrics(metrics)
	assert.Equal(t, 1, len(errors))
	assert.Error(t, errors[0])
}

// TestValidateZoneMetricsInvalidPower tests metrics validation with invalid power
func TestValidateZoneMetricsInvalidPower(t *testing.T) {
	t.Parallel()

	badPower := float32(101.0) // Out of range
	metrics := &ZoneMetrics{
		HeatingPowerPercentage: &badPower,
	}

	errors := ValidateZoneMetrics(metrics)
	assert.Equal(t, 1, len(errors))
	assert.Error(t, errors[0])
}

// TestValidateZoneMetricsMultipleErrors tests metrics validation with multiple errors
func TestValidateZoneMetricsMultipleErrors(t *testing.T) {
	t.Parallel()

	badTemp := float32(-100.0)    // Out of range
	badHumidity := float32(-10.0) // Out of range
	badPower := float32(150.0)    // Out of range

	metrics := &ZoneMetrics{
		MeasuredTemperatureCelsius: &badTemp,
		MeasuredHumidity:           &badHumidity,
		HeatingPowerPercentage:     &badPower,
	}

	errors := ValidateZoneMetrics(metrics)
	assert.Equal(t, 3, len(errors))
}

// TestValidationErrorError tests ValidationError.Error() method
func TestValidationErrorError(t *testing.T) {
	t.Parallel()

	err := &ValidationError{
		Field:  "temperature",
		Value:  100.0,
		Reason: "outside valid range",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "validation error")
	assert.Contains(t, errorMsg, "temperature")
	assert.Contains(t, errorMsg, "100")
}
