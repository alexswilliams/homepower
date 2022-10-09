package kasa

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"strconv"
	"time"
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
	updateInfoMetric               func(status *periodicDeviceReport) error
	updateActiveMode               func(status *periodicDeviceReport) error
	wifiRssi                       *prometheus.Gauge
	deviceTurnedOn                 *prometheus.Gauge
	ledTurnedOn                    *prometheus.Gauge                        // only for switches
	onTime                         *prometheus.Gauge                        // only for switches
	isUpdating                     *prometheus.Gauge                        // only for switches
	updateLightMode                func(status *periodicDeviceReport) error // only for lights
	brightness                     *prometheus.Gauge                        // only for lights
	colourTemperature              *prometheus.Gauge                        // only for lights
	hue                            *prometheus.Gauge                        // only for lights
	saturation                     *prometheus.Gauge                        // only for lights
	powerMilliWatts                *prometheus.Gauge                        // HS110, KL50, KL110, KL130,
	voltageMilliVolts              *prometheus.Gauge                        // HS110 only
	currentMilliAmps               *prometheus.Gauge                        // HS110 only
	totalWattHours                 *prometheus.Gauge                        // HS110, KL50, and maybe KL130
}

func registerMetrics(registry prometheus.Registerer, config *types.DeviceConfig) prometheusMetrics {
	commonLabels := types.GenerateCommonLabels(config)
	metrics := prometheusMetrics{
		isSwitch:                       isSwitch(config),
		isLight:                        isLight(config),
		isVariableTemperature:          isLightVariableTemperature(config),
		isColoured:                     isLightColoured(config),
		hasPowerMonitoring:             hasPowerMonitoring(config),
		hasTotalEnergyMonitoring:       hasTotalEnergyMonitoring(config),
		hasCurrentAndVoltageMonitoring: hasCurrentAndVoltageMonitoring(config),

		commonLabels:     commonLabels,
		updateInfoMetric: registerInfoMetricUpdater(registry, commonLabels, isLight(config)),
		updateActiveMode: registerModeMetricUpdater(registry, commonLabels, "active_mode", "mode", func(report *periodicDeviceReport) string {
			return report.ActiveMode
		}),
		wifiRssi:       newGauge(registry, commonLabels, "wifi_rssi_db"),
		deviceTurnedOn: newGauge(registry, commonLabels, "device_turned_on_bool"),
	}
	if metrics.isSwitch {
		metrics.ledTurnedOn = newGauge(registry, commonLabels, "led_turned_on_bool")
		metrics.onTime = newGauge(registry, commonLabels, "switched_on_time_seconds")
		metrics.isUpdating = newGauge(registry, commonLabels, "is_updating_bool")
	}
	if metrics.isLight {
		metrics.updateLightMode = registerModeMetricUpdater(registry, commonLabels, "bulb_mode", "mode", func(report *periodicDeviceReport) string {
			return report.smartBulbInfo.Mode
		})
		metrics.brightness = newGauge(registry, commonLabels, "bulb_brightness_percent")
		if metrics.isVariableTemperature {
			metrics.colourTemperature = newGauge(registry, commonLabels, "bulb_colour_temperature_kelvin")
		}
		if metrics.isColoured {
			metrics.hue = newGauge(registry, commonLabels, "bulb_hue")
			metrics.saturation = newGauge(registry, commonLabels, "bulb_saturation_percent")
		}
	}
	if metrics.hasPowerMonitoring {
		metrics.powerMilliWatts = newGauge(registry, commonLabels, "em_power_mw")
	}
	if metrics.hasTotalEnergyMonitoring {
		metrics.totalWattHours = newGauge(registry, commonLabels, "em_total_energy_wh")
	}
	if metrics.hasCurrentAndVoltageMonitoring {
		metrics.currentMilliAmps = newGauge(registry, commonLabels, "em_current_ma")
		metrics.voltageMilliVolts = newGauge(registry, commonLabels, "em_voltage_mv")
	}
	metrics.resetToRogueValues()
	return metrics
}

func newGauge(registry prometheus.Registerer, commonLabels prometheus.Labels, name string) *prometheus.Gauge {
	var gauge = prometheus.NewGauge(prometheus.GaugeOpts{Name: name, ConstLabels: commonLabels, Namespace: "kasa"})
	registry.MustRegister(gauge)
	return &gauge
}
func setIfPresent(gauge *prometheus.Gauge, value float64) {
	if gauge != nil {
		(*gauge).Set(value)
	}
}
func setFromBool(gauge *prometheus.Gauge, value bool) {
	if value {
		setIfPresent(gauge, 1.0)
	} else {
		setIfPresent(gauge, 0.0)
	}
}
func setFromInt(gauge *prometheus.Gauge, value int) {
	setIfPresent(gauge, float64(value))
}
func setFromDurationAsSeconds(gauge *prometheus.Gauge, value time.Duration) {
	setIfPresent(gauge, value.Seconds())
}

func (metrics *prometheusMetrics) updateMetrics(status *periodicDeviceReport) error {
	if status == nil {
		metrics.resetToRogueValues()
	} else {
		setFromInt(metrics.wifiRssi, status.WifiRssi)
		if metrics.isSwitch && status.smartPlugInfo != nil {
			setFromBool(metrics.deviceTurnedOn, status.RelayOn)
			setFromBool(metrics.ledTurnedOn, status.LedOn)
			setFromDurationAsSeconds(metrics.onTime, status.OnTime)
			setFromBool(metrics.isUpdating, status.Updating)
		}
		if metrics.isLight && status.smartBulbInfo != nil {
			setFromBool(metrics.deviceTurnedOn, status.IsOn)
			setFromInt(metrics.brightness, status.Brightness)
			if metrics.isVariableTemperature {
				setFromInt(metrics.colourTemperature, status.ColourTemperature)
			}
			if metrics.isColoured {
				setFromInt(metrics.hue, status.Hue)
				setFromInt(metrics.saturation, status.Saturation)
			}
		}
		if metrics.hasPowerMonitoring {
			setFromInt(metrics.powerMilliWatts, status.PowerMilliWatts)
		}
		if metrics.hasTotalEnergyMonitoring {
			setFromInt(metrics.totalWattHours, status.TotalEnergyWattHours)
		}
		if metrics.hasCurrentAndVoltageMonitoring {
			setFromInt(metrics.currentMilliAmps, status.CurrentMilliAmps)
			setFromInt(metrics.voltageMilliVolts, status.VoltageMilliVolts)
		}
		if err := metrics.updateInfoMetric(status); err != nil {
			return fmt.Errorf("could not update info metric: %w", err)
		}
		if err := metrics.updateActiveMode(status); err != nil {
			return fmt.Errorf("could not update active mode metric: %w", err)
		}
		if metrics.updateLightMode != nil {
			if err := metrics.updateLightMode(status); err != nil {
				return fmt.Errorf("could not update active mode metric: %w", err)
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
	setIfPresent(metrics.wifiRssi, +1.0) // nb: positive rogue value
	setIfPresent(metrics.deviceTurnedOn, -1.0)
	setIfPresent(metrics.ledTurnedOn, -1.0)
	setIfPresent(metrics.onTime, -1.0)
	setIfPresent(metrics.isUpdating, -1.0)
	setIfPresent(metrics.brightness, -1.0)
	setIfPresent(metrics.colourTemperature, -1.0)
	setIfPresent(metrics.hue, -1.0)
	setIfPresent(metrics.saturation, -1.0)
	setIfPresent(metrics.powerMilliWatts, -1.0)
	setIfPresent(metrics.voltageMilliVolts, -1.0)
	setIfPresent(metrics.currentMilliAmps, -1.0)
	setIfPresent(metrics.totalWattHours, -1.0)
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
