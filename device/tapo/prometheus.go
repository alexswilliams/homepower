package tapo

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type prometheusMetrics struct {
	isLight              bool
	isSwitch             bool
	hasEnergyMonitoring  bool
	commonLabels         prometheus.Labels
	updateInfoMetric     func(status *deviceStatus) error
	overheated           *prometheus.Gauge
	wifiRssi             *prometheus.Gauge
	signalLevel          *prometheus.Gauge
	deviceTurnedOn       *prometheus.Gauge
	onTime               *prometheus.Gauge // only for switches
	brightness           *prometheus.Gauge // only for lights
	colourTemperature    *prometheus.Gauge // only for lights
	hue                  *prometheus.Gauge // only for lights
	saturation           *prometheus.Gauge // only for lights
	powerMilliWatts      *prometheus.Gauge // only for P110
	monthEnergyWattHours *prometheus.Gauge // only for P110
	todayEnergyWattHours *prometheus.Gauge // only for P110
}

func registerMetrics(registry prometheus.Registerer, commonLabels prometheus.Labels, isSwitch, isLight, hasEnergyMonitoring bool) prometheusMetrics {
	metrics := prometheusMetrics{
		isLight:             isLight,
		isSwitch:            isSwitch,
		hasEnergyMonitoring: hasEnergyMonitoring,
		commonLabels:        commonLabels,
		updateInfoMetric:    registerInfoMetricUpdater(registry, commonLabels),
		overheated:          newGauge(registry, commonLabels, "overheated_bool"),
		wifiRssi:            newGauge(registry, commonLabels, "wifi_rssi_db"),
		signalLevel:         newGauge(registry, commonLabels, "signal_level"),
		deviceTurnedOn:      newGauge(registry, commonLabels, "device_turned_on_bool"),
	}
	if isSwitch {
		metrics.onTime = newGauge(registry, commonLabels, "switched_on_time_seconds")
	}
	if isLight {
		metrics.brightness = newGauge(registry, commonLabels, "bulb_brightness_percent")
		metrics.colourTemperature = newGauge(registry, commonLabels, "bulb_colour_temperature_kelvin")
		metrics.hue = newGauge(registry, commonLabels, "bulb_hue")
		metrics.saturation = newGauge(registry, commonLabels, "bulb_saturation_percent")
	}
	if hasEnergyMonitoring {
		metrics.powerMilliWatts = newGauge(registry, commonLabels, "em_power_mw")
		metrics.todayEnergyWattHours = newGauge(registry, commonLabels, "em_today_energy_wh")
		metrics.monthEnergyWattHours = newGauge(registry, commonLabels, "em_month_energy_wh")
	}
	metrics.resetToRogueValues()
	return metrics
}

func newGauge(registry prometheus.Registerer, commonLabels prometheus.Labels, name string) *prometheus.Gauge {
	var gauge = prometheus.NewGauge(prometheus.GaugeOpts{Name: name, ConstLabels: commonLabels, Namespace: "tapo"})
	registry.MustRegister(gauge)
	return &gauge
}

func (metrics *prometheusMetrics) updateMetrics(status *deviceStatus) error {
	if status == nil {
		metrics.resetToRogueValues()
	} else {
		setFromBool(metrics.overheated, status.Overheated)
		setFromInt(metrics.wifiRssi, status.WifiRssi)
		setFromInt(metrics.signalLevel, status.SignalLevel)
		if metrics.isSwitch && status.smartPlugInfo != nil {
			setFromBool(metrics.deviceTurnedOn, status.RelayOn)
			setFromDurationAsSeconds(metrics.onTime, status.OnTime)
		}
		if metrics.isLight && status.smartBulbInfo != nil {
			setFromBool(metrics.deviceTurnedOn, status.LightOn)
			setFromInt(metrics.brightness, status.Brightness)
			setFromInt(metrics.colourTemperature, status.ColourTemperature)
			setFromInt(metrics.hue, status.Hue)
			setFromInt(metrics.saturation, status.Saturation)
		}
		if metrics.hasEnergyMonitoring && status.energyMeterInfo != nil {
			setFromInt(metrics.powerMilliWatts, status.PowerMilliWatts)
			setFromInt(metrics.monthEnergyWattHours, status.MonthEnergyWattHours)
			setFromInt(metrics.todayEnergyWattHours, status.TodayEnergyWattHours)
		}
		if err := metrics.updateInfoMetric(status); err != nil {
			return fmt.Errorf("could not update info metric: %w", err)
		}
	}
	return nil
}

func (metrics *prometheusMetrics) resetToRogueValues() {
	_ = metrics.updateInfoMetric(nil)
	setIfPresent(metrics.overheated, -1.0)
	setIfPresent(metrics.wifiRssi, +1.0) // nb: positive rogue value
	setIfPresent(metrics.signalLevel, -1.0)
	setIfPresent(metrics.deviceTurnedOn, -1.0)
	setIfPresent(metrics.onTime, -1.0)
	setIfPresent(metrics.brightness, -1.0)
	setIfPresent(metrics.colourTemperature, -1.0)
	setIfPresent(metrics.hue, -1.0)
	setIfPresent(metrics.saturation, -1.0)
	setIfPresent(metrics.powerMilliWatts, -1.0)
	setIfPresent(metrics.monthEnergyWattHours, -1.0)
	setIfPresent(metrics.todayEnergyWattHours, -1.0)
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
