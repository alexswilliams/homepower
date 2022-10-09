package tapo

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
)

type prometheusMetrics struct {
	isLight             bool
	isSwitch            bool
	hasEnergyMonitoring bool
	commonLabels        prometheus.Labels

	updateInfoMetric func(status *deviceStatus) error
	overheated       *prometheus.Gauge
	wifiRssi         *prometheus.Gauge
	signalLevel      *prometheus.Gauge
	deviceTurnedOn   *prometheus.Gauge

	onTime *prometheus.Gauge // only for switches

	brightness        *prometheus.Gauge // only for lights
	colourTemperature *prometheus.Gauge // only for lights
	hue               *prometheus.Gauge // only for lights
	saturation        *prometheus.Gauge // only for lights

	powerMilliWatts      *prometheus.Gauge // P110
	monthEnergyWattHours *prometheus.Gauge // P110
	todayEnergyWattHours *prometheus.Gauge // P110
}

func registerMetrics(registry prometheus.Registerer, commonLabels prometheus.Labels, isSwitch, isLight, hasEnergyMonitoring bool) *prometheusMetrics {
	metrics := prometheusMetrics{
		isLight:             isLight,
		isSwitch:            isSwitch,
		hasEnergyMonitoring: hasEnergyMonitoring,
		commonLabels:        commonLabels,

		updateInfoMetric: registerInfoMetricUpdater(registry, commonLabels),
		overheated:       types.NewGauge(registry, commonLabels, "tapo", "overheated_bool"),
		wifiRssi:         types.NewGauge(registry, commonLabels, "tapo", "wifi_rssi_db"),
		signalLevel:      types.NewGauge(registry, commonLabels, "tapo", "signal_level"),
		deviceTurnedOn:   types.NewGauge(registry, commonLabels, "tapo", "device_turned_on_bool"),
	}
	if isSwitch {
		metrics.onTime = types.NewGauge(registry, commonLabels, "tapo", "switched_on_time_seconds")
	}
	if isLight {
		metrics.brightness = types.NewGauge(registry, commonLabels, "tapo", "bulb_brightness_percent")
		metrics.colourTemperature = types.NewGauge(registry, commonLabels, "tapo", "bulb_colour_temperature_kelvin")
		metrics.hue = types.NewGauge(registry, commonLabels, "tapo", "bulb_hue")
		metrics.saturation = types.NewGauge(registry, commonLabels, "tapo", "bulb_saturation_percent")
	}
	if hasEnergyMonitoring {
		metrics.powerMilliWatts = types.NewGauge(registry, commonLabels, "tapo", "em_power_mw")
		metrics.todayEnergyWattHours = types.NewGauge(registry, commonLabels, "tapo", "em_today_energy_wh")
		metrics.monthEnergyWattHours = types.NewGauge(registry, commonLabels, "tapo", "em_month_energy_wh")
	}
	metrics.resetToRogueValues()
	return &metrics
}

func (metrics *prometheusMetrics) updateMetrics(status *deviceStatus) error {
	if status == nil {
		metrics.resetToRogueValues()
	} else {
		types.SetFromBool(metrics.overheated, status.Overheated)
		types.SetFromInt(metrics.wifiRssi, status.WifiRssi)
		types.SetFromInt(metrics.signalLevel, status.SignalLevel)
		if metrics.isSwitch && status.smartPlugInfo != nil {
			types.SetFromBool(metrics.deviceTurnedOn, status.RelayOn)
			types.SetFromDurationAsSeconds(metrics.onTime, status.OnTime)
		}
		if metrics.isLight && status.smartBulbInfo != nil {
			types.SetFromBool(metrics.deviceTurnedOn, status.LightOn)
			types.SetFromInt(metrics.brightness, status.Brightness)
			types.SetFromInt(metrics.colourTemperature, status.ColourTemperature)
			types.SetFromInt(metrics.hue, status.Hue)
			types.SetFromInt(metrics.saturation, status.Saturation)
		}
		if metrics.hasEnergyMonitoring && status.energyMeterInfo != nil {
			types.SetFromInt(metrics.powerMilliWatts, status.PowerMilliWatts)
			types.SetFromInt(metrics.monthEnergyWattHours, status.MonthEnergyWattHours)
			types.SetFromInt(metrics.todayEnergyWattHours, status.TodayEnergyWattHours)
		}
		if err := metrics.updateInfoMetric(status); err != nil {
			return fmt.Errorf("could not update info metric: %w", err)
		}
	}
	return nil
}

func (metrics *prometheusMetrics) resetToRogueValues() {
	_ = metrics.updateInfoMetric(nil)
	types.SetIfPresent(metrics.overheated, -1.0)
	types.SetIfPresent(metrics.wifiRssi, +1.0) // nb: positive rogue value
	types.SetIfPresent(metrics.signalLevel, -1.0)
	types.SetIfPresent(metrics.deviceTurnedOn, -1.0)
	types.SetIfPresent(metrics.onTime, -1.0)
	types.SetIfPresent(metrics.brightness, -1.0)
	types.SetIfPresent(metrics.colourTemperature, -1.0)
	types.SetIfPresent(metrics.hue, -1.0)
	types.SetIfPresent(metrics.saturation, -1.0)
	types.SetIfPresent(metrics.powerMilliWatts, -1.0)
	types.SetIfPresent(metrics.monthEnergyWattHours, -1.0)
	types.SetIfPresent(metrics.todayEnergyWattHours, -1.0)
}

func registerInfoMetricUpdater(registry prometheus.Registerer, commonLabels prometheus.Labels) func(status *deviceStatus) error {
	var infoMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "device_info",
		Namespace:   "tapo",
		ConstLabels: commonLabels,
	}, []string{
		"alias", "device_id", "firmware_version", "hardware_id", "mac_address", "model_name", "oem_id", "device_type",
	})
	registry.MustRegister(infoMetric)
	return func(status *deviceStatus) error {
		infoMetric.Reset()
		if status != nil {
			metricWithLabelValues, err := infoMetric.GetMetricWith(prometheus.Labels{
				"alias":            status.Alias,
				"device_id":        status.DeviceId,
				"firmware_version": status.FirmwareVersion,
				"hardware_id":      status.HardwareId,
				"mac_address":      status.Mac,
				"model_name":       status.ModelName,
				"oem_id":           status.OemId,
				"device_type":      status.DeviceType,
			})
			if err != nil {
				return fmt.Errorf("could not generate label values for info metric: %w", err)
			}
			metricWithLabelValues.Set(1.0)
		}
		return nil
	}
}
