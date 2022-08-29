package kasa

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"strconv"
	"strings"
)

type LatestDeviceReport struct {
	Latest *PeriodicDeviceReport // nil indicates no scrape has happened, or the scrape failed
}

func supportsEMeter(model types.DeviceType) bool {
	switch model {
	case types.KasaHS110, types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return true
	default:
		return false
	}
}

func isLight(model types.DeviceType) bool {
	return model == types.KasaKL50B || model == types.KasaKL110B || model == types.KasaKL130B
}

func RegisterMetrics(registry prometheus.Registerer, dev types.DeviceConfig, lastDeviceReport *LatestDeviceReport) *prometheus.GaugeVec {
	var constLabels = GenerateCommonLabels(dev)

	if supportsEMeter(dev.Model) {
		registerEMeterMetrics(registry, lastDeviceReport, constLabels)
	}
	if isLight(dev.Model) {
		registerLightMetrics(registry, lastDeviceReport, constLabels, dev.Model == types.KasaKL130B, dev.Model == types.KasaKL130B)
	} else {
		registerSwitchMetrics(registry, lastDeviceReport, constLabels)
	}

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_wifi_rssi", ConstLabels: constLabels}, func() float64 {
		if lastDeviceReport.Latest == nil {
			return +1.0 // rssi is always negative, so this indicates a scrape failure
		} else {
			return float64(lastDeviceReport.Latest.WifiSignalStrength)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_last_scrape_duration_ms", ConstLabels: constLabels}, func() float64 {
		if lastDeviceReport.Latest == nil {
			return -1.0 // durations are always positive (mumble, leap seconds), so this indicates a scrape failure
		} else {
			return float64(lastDeviceReport.Latest.ScrapeDuration.Milliseconds())
		}
	}))

	var infoMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:        "kasa_dev_info",
		ConstLabels: constLabels,
	}, []string{
		"kasa_active_mode",
		"kasa_alias",
		"kasa_model_description",
		"kasa_device_id",
		"kasa_hardware_id",
		"kasa_dev_mac",
		"kasa_model_name",
		"kasa_dev_type",
		"kasa_light_device_state",
		"kasa_light_is_dimmable",
		"kasa_light_is_colour",
		"kasa_light_is_variable_temp",
		"kasa_light_on_mode",
		"kasa_light_beam_angle",
		"kasa_light_min_voltage",
		"kasa_light_max_voltage",
		"kasa_light_wattage",
		"kasa_light_incandescent_equiv",
		"kasa_light_max_lumens",
	})
	registry.MustRegister(infoMetric)
	return infoMetric
}

func GenerateCommonLabels(dev types.DeviceConfig) prometheus.Labels {
	return prometheus.Labels{
		"dev_room":      dev.Room,
		"dev_name":      dev.Name,
		"dev_ip":        dev.Ip,
		"dev_full_name": strings.TrimSpace(fmt.Sprintf("%s %s", dev.Room, dev.Name)),
		"is_light":      strconv.FormatBool(isLight(dev.Model)),
	}
}

func registerSwitchMetrics(registry prometheus.Registerer, report *LatestDeviceReport, labels prometheus.Labels) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_dev_is_on", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return -1.0 // normal values are 0.0 or 1.0
		} else {
			return boolToFloat(report.Latest.SmartPlugSwitchInfo.RelayOn)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_led_is_on", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return -1.0 // normal values are 0.0 or 1.0
		} else {
			return boolToFloat(report.Latest.SmartPlugSwitchInfo.LedOn)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_on_time_seconds", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return -1.0 // i'm hoping durations as reported by the device are always positive, so this is a rogue value
		} else {
			return report.Latest.SmartPlugSwitchInfo.OnTime.Seconds()
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_updating", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return -1.0 // normal values are 0.0 or 1.0
		} else {
			return boolToFloat(report.Latest.SmartPlugSwitchInfo.Updating)
		}
	}))
}

func registerLightMetrics(registry prometheus.Registerer, report *LatestDeviceReport, labels prometheus.Labels, isColour bool, isVariableTemp bool) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_dev_is_on", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil {
			return -1.0 // normal values are 0.0 or 1.0
		} else {
			return boolToFloat(report.Latest.SmartLightInfo.IsOn)
		}
	}))
	if isColour {
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_hue", ConstLabels: labels}, func() float64 {
			if report.Latest == nil || report.Latest.SmartLightInfo == nil {
				return -1.0 // this indicates a scrape failure
			} else if !report.Latest.SmartLightInfo.IsColour {
				return -2.0 // this indicates the device does not support colour
			} else {
				return float64(report.Latest.SmartLightInfo.Hue)
			}
		}))
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_saturation", ConstLabels: labels}, func() float64 {
			if report.Latest == nil || report.Latest.SmartLightInfo == nil {
				return -1.0 // this indicates a scrape failure
			} else if !report.Latest.SmartLightInfo.IsColour {
				return -2.0 // this indicates the device does not support colour
			} else {
				return float64(report.Latest.SmartLightInfo.Saturation)
			}
		}))
	}
	if isVariableTemp {
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_colour_temperature", ConstLabels: labels}, func() float64 {
			if report.Latest == nil || report.Latest.SmartLightInfo == nil {
				return -1.0 // this indicates a scrape failure
			} else if !report.Latest.SmartLightInfo.IsVariableColourTemperature {
				return -2.0 // this indicates the device does not support variable colour temperatures
			} else {
				return float64(report.Latest.SmartLightInfo.ColourTemperature)
			}
		}))
	}
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_brightness", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil {
			return -1.0
		} else {
			return float64(report.Latest.SmartLightInfo.Brightness)
		}
	}))
}

func boolToFloat(on bool) float64 {
	if on {
		return 1.0
	} else {
		return 0.0
	}
}

func registerEMeterMetrics(registry prometheus.Registerer, report *LatestDeviceReport, labels prometheus.Labels) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_current_ma", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return -1.0
		} else {
			return float64(report.Latest.EnergyMeterInfo.CurrentMilliAmps)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_power_mw", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return -1.0
		} else {
			return float64(report.Latest.EnergyMeterInfo.PowerMilliWatts)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_voltage_mv", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return -1.0
		} else {
			return float64(report.Latest.EnergyMeterInfo.VoltageMilliVolts)
		}
	}))
	registry.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{Name: "kasa_totalEnergy_wh", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return -1.0
		} else {
			return float64(report.Latest.EnergyMeterInfo.TotalEnergyWattHours)
		}
	}))
}
