package kasa

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"strconv"
)

type prometheusMetrics struct {
	isSwitch                       bool
	isLight                        bool
	isVariableTemperature          bool
	isColoured                     bool
	hasPowerMonitoring             bool
	hasTotalEnergyMonitoring       bool
	hasCurrentAndVoltageMonitoring bool
	commonLabels                   prometheus.Labels

	updateInfoMetric func(status *periodicDeviceReport) error
	updateActiveMode func(status *periodicDeviceReport) error
	wifiRssi         *prometheus.Gauge
	deviceTurnedOn   *prometheus.Gauge

	ledTurnedOn *prometheus.Gauge // only for switches
	onTime      *prometheus.Gauge // only for switches
	isUpdating  *prometheus.Gauge // only for switches

	updateLightMode   func(status *periodicDeviceReport) error // only for lights
	brightness        *prometheus.Gauge                        // only for lights
	colourTemperature *prometheus.Gauge                        // only for lights
	hue               *prometheus.Gauge                        // only for lights
	saturation        *prometheus.Gauge                        // only for lights

	powerMilliWatts   *prometheus.Gauge // HS110, KL50, KL110, KL130,
	voltageMilliVolts *prometheus.Gauge // HS110 only
	currentMilliAmps  *prometheus.Gauge // HS110 only
	totalWattHours    *prometheus.Gauge // HS110, KL50, and maybe KL130
}

func registerMetrics(registry prometheus.Registerer, config *types.DeviceConfig) *prometheusMetrics {
	commonLabels := types.GenerateCommonLabels(config)
	metrics := prometheusMetrics{
		isSwitch:                       isSwitch(config),
		isLight:                        isLight(config),
		isVariableTemperature:          isLightVariableTemperature(config),
		isColoured:                     isLightColoured(config),
		hasPowerMonitoring:             hasPowerMonitoring(config),
		hasTotalEnergyMonitoring:       hasTotalEnergyMonitoring(config),
		hasCurrentAndVoltageMonitoring: hasCurrentAndVoltageMonitoring(config),
		commonLabels:                   commonLabels,

		wifiRssi:         types.NewGauge(registry, commonLabels, "kasa", "wifi_rssi_db"),
		deviceTurnedOn:   types.NewGauge(registry, commonLabels, "kasa", "device_turned_on_bool"),
		updateInfoMetric: registerInfoMetricUpdater(registry, commonLabels, isLight(config)),
		updateActiveMode: registerModeMetricUpdater(registry, commonLabels, "active_mode", "mode",
			func(report *periodicDeviceReport) string {
				return report.ActiveMode
			}),
	}
	if metrics.isSwitch {
		metrics.ledTurnedOn = types.NewGauge(registry, commonLabels, "kasa", "led_turned_on_bool")
		metrics.onTime = types.NewGauge(registry, commonLabels, "kasa", "switched_on_time_seconds")
		metrics.isUpdating = types.NewGauge(registry, commonLabels, "kasa", "is_updating_bool")
	}
	if metrics.isLight {
		metrics.brightness = types.NewGauge(registry, commonLabels, "kasa", "bulb_brightness_percent")
		metrics.updateLightMode = registerModeMetricUpdater(registry, commonLabels, "bulb_mode", "mode",
			func(report *periodicDeviceReport) string {
				return report.smartBulbInfo.Mode
			})
		if metrics.isVariableTemperature {
			metrics.colourTemperature = types.NewGauge(registry, commonLabels, "kasa", "bulb_colour_temperature_kelvin")
		}
		if metrics.isColoured {
			metrics.hue = types.NewGauge(registry, commonLabels, "kasa", "bulb_hue")
			metrics.saturation = types.NewGauge(registry, commonLabels, "kasa", "bulb_saturation_percent")
		}
	}
	if metrics.hasPowerMonitoring {
		metrics.powerMilliWatts = types.NewGauge(registry, commonLabels, "kasa", "em_power_mw")
	}
	if metrics.hasTotalEnergyMonitoring {
		metrics.totalWattHours = types.NewGauge(registry, commonLabels, "kasa", "em_total_energy_wh")
	}
	if metrics.hasCurrentAndVoltageMonitoring {
		metrics.currentMilliAmps = types.NewGauge(registry, commonLabels, "kasa", "em_current_ma")
		metrics.voltageMilliVolts = types.NewGauge(registry, commonLabels, "kasa", "em_voltage_mv")
	}
	metrics.resetToRogueValues()
	return &metrics
}

func (metrics *prometheusMetrics) updateMetrics(status *periodicDeviceReport) error {
	if status == nil {
		metrics.resetToRogueValues()
	} else {
		types.SetFromInt(metrics.wifiRssi, status.WifiRssi)
		if metrics.isSwitch && status.smartPlugInfo != nil {
			types.SetFromBool(metrics.deviceTurnedOn, status.RelayOn)
			types.SetFromBool(metrics.ledTurnedOn, status.LedOn)
			types.SetFromDurationAsSeconds(metrics.onTime, status.OnTime)
			types.SetFromBool(metrics.isUpdating, status.Updating)
		}
		if metrics.isLight && status.smartBulbInfo != nil {
			types.SetFromBool(metrics.deviceTurnedOn, status.IsOn)
			types.SetFromInt(metrics.brightness, status.Brightness)
			if metrics.isVariableTemperature {
				types.SetFromInt(metrics.colourTemperature, status.ColourTemperature)
			}
			if metrics.isColoured {
				types.SetFromInt(metrics.hue, status.Hue)
				types.SetFromInt(metrics.saturation, status.Saturation)
			}
		}
		if metrics.hasPowerMonitoring {
			types.SetFromInt(metrics.powerMilliWatts, status.PowerMilliWatts)
		}
		if metrics.hasTotalEnergyMonitoring {
			types.SetFromInt(metrics.totalWattHours, status.TotalEnergyWattHours)
		}
		if metrics.hasCurrentAndVoltageMonitoring {
			types.SetFromInt(metrics.currentMilliAmps, status.CurrentMilliAmps)
			types.SetFromInt(metrics.voltageMilliVolts, status.VoltageMilliVolts)
		}
		if err := metrics.updateInfoMetric(status); err != nil {
			return fmt.Errorf("could not update info metric: %w", err)
		}
		if err := metrics.updateActiveMode(status); err != nil {
			return fmt.Errorf("could not update active mode metric: %w", err)
		}
		if metrics.updateLightMode != nil {
			if err := metrics.updateLightMode(status); err != nil {
				return fmt.Errorf("could not update light mode metric: %w", err)
			}
		}
	}
	return nil
}

func (metrics *prometheusMetrics) resetToRogueValues() {
	_ = metrics.updateInfoMetric(nil)
	_ = metrics.updateActiveMode(nil)
	if metrics.updateLightMode != nil {
		_ = metrics.updateLightMode(nil)
	}
	types.SetIfPresent(metrics.wifiRssi, +1.0) // nb: positive rogue value
	types.SetIfPresent(metrics.deviceTurnedOn, -1.0)
	types.SetIfPresent(metrics.ledTurnedOn, -1.0)
	types.SetIfPresent(metrics.onTime, -1.0)
	types.SetIfPresent(metrics.isUpdating, -1.0)
	types.SetIfPresent(metrics.brightness, -1.0)
	types.SetIfPresent(metrics.colourTemperature, -1.0)
	types.SetIfPresent(metrics.hue, -1.0)
	types.SetIfPresent(metrics.saturation, -1.0)
	types.SetIfPresent(metrics.powerMilliWatts, -1.0)
	types.SetIfPresent(metrics.voltageMilliVolts, -1.0)
	types.SetIfPresent(metrics.currentMilliAmps, -1.0)
	types.SetIfPresent(metrics.totalWattHours, -1.0)
}

func registerInfoMetricUpdater(registry prometheus.Registerer, commonLabels prometheus.Labels, isLight bool) func(status *periodicDeviceReport) error {
	var infoMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "device_info", Namespace: "kasa", ConstLabels: commonLabels}, []string{
		"alias", "device_id", "firmware_version", "hardware_id", "mac_address", "model_name", "model_description", "oem_id", "device_type",
	})
	registry.MustRegister(infoMetric)
	var bulbInfoMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "bulb_info", Namespace: "kasa", ConstLabels: commonLabels}, []string{
		"is_dimmable", "is_colour", "is_variable_temp", "beam_angle", "min_voltage", "max_voltage", "wattage", "incandescent_equiv", "max_lumens",
	})
	if isLight {
		registry.MustRegister(bulbInfoMetric)
	}
	return func(status *periodicDeviceReport) error {
		infoMetric.Reset()
		bulbInfoMetric.Reset()
		if status != nil {
			metricWithLabelValues, err := infoMetric.GetMetricWith(prometheus.Labels{
				"alias":             status.Alias,
				"device_id":         status.DeviceId,
				"firmware_version":  status.SoftwareVersion,
				"hardware_id":       status.HardwareId,
				"mac_address":       status.Mac,
				"model_name":        status.ModelName,
				"model_description": status.ModelDescription,
				"oem_id":            status.OemId,
				"device_type":       status.DeviceType,
			})
			if err != nil {
				return fmt.Errorf("could not generate label values for info metric: %w", err)
			}
			metricWithLabelValues.Set(1.0)
			if isLight {
				bulbMetricWithLabelValues, err := bulbInfoMetric.GetMetricWith(prometheus.Labels{
					"is_dimmable":        strconv.FormatBool(status.IsDimmable),
					"is_colour":          strconv.FormatBool(status.IsColour),
					"is_variable_temp":   strconv.FormatBool(status.IsVariableColourTemperature),
					"beam_angle":         strconv.Itoa(status.LampBeamAngle),
					"min_voltage":        strconv.Itoa(status.MinimumVoltage),
					"max_voltage":        strconv.Itoa(status.MaximumVoltage),
					"wattage":            strconv.Itoa(status.Wattage),
					"incandescent_equiv": strconv.Itoa(status.IncandescentEquivalent),
					"max_lumens":         strconv.Itoa(status.MaximumLumens),
				})
				if err != nil {
					return fmt.Errorf("could not generate label values for bulb metric: %w", err)
				}
				bulbMetricWithLabelValues.Set(1.0)
			}
		}
		return nil
	}
}

func registerModeMetricUpdater(registry prometheus.Registerer, commonLabels prometheus.Labels, name string, labelName string, jsonKey func(report *periodicDeviceReport) string) func(status *periodicDeviceReport) error {
	var metric = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Namespace: "kasa", ConstLabels: commonLabels}, []string{labelName})
	registry.MustRegister(metric)
	return func(status *periodicDeviceReport) error {
		metric.Reset()
		if status != nil {
			metricWithLabelValues, err := metric.GetMetricWith(prometheus.Labels{labelName: jsonKey(status)})
			if err != nil {
				return fmt.Errorf("could not generate label values for info metric: %w", err)
			}
			metricWithLabelValues.Set(1.0)
		}
		return nil
	}
}
