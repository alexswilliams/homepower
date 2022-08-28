package kasa

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"math"
	"strconv"
	"strings"
)

type LatestDeviceReport struct {
	Latest *PeriodicDeviceReport
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
	var constLabels prometheus.Labels = map[string]string{
		"dev_room":      dev.Room,
		"dev_name":      dev.Name,
		"dev_ip":        dev.Ip,
		"dev_full_name": strings.TrimSpace(fmt.Sprintf("%s %s", dev.Room, dev.Name)),
		"is_light":      strconv.FormatBool(isLight(dev.Model)),
	}

	if supportsEMeter(dev.Model) {
		registerEMeterMetrics(registry, lastDeviceReport, constLabels)
	}
	if isLight(dev.Model) {
		registerLightMetrics(registry, lastDeviceReport, constLabels)
	} else {
		registerSwitchMetrics(registry, lastDeviceReport, constLabels)
	}

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_wifi_rssi", ConstLabels: constLabels}, func() float64 {
		if lastDeviceReport.Latest == nil {
			return math.NaN()
		} else {
			return float64(lastDeviceReport.Latest.WifiSignalStrength)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_scrape_duration_ms", ConstLabels: constLabels}, func() float64 {
		if lastDeviceReport.Latest == nil {
			return math.NaN()
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
	})
	registry.MustRegister(infoMetric)
	return infoMetric
}

func registerSwitchMetrics(registry prometheus.Registerer, report *LatestDeviceReport, labels prometheus.Labels) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_dev_is_on", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return math.NaN()
		} else {
			return boolToFloat(report.Latest.SmartPlugSwitchInfo.RelayOn)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_led_is_on", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return math.NaN()
		} else {
			return boolToFloat(report.Latest.SmartPlugSwitchInfo.LedOn)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_on_time_seconds", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return math.NaN()
		} else {
			return report.Latest.SmartPlugSwitchInfo.OnTime.Seconds()
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_updating", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartPlugSwitchInfo == nil {
			return math.NaN()
		} else {
			return boolToFloat(report.Latest.SmartPlugSwitchInfo.Updating)
		}
	}))
}

func registerLightMetrics(registry prometheus.Registerer, report *LatestDeviceReport, labels prometheus.Labels) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_dev_is_on", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil {
			return math.NaN()
		} else {
			return boolToFloat(report.Latest.SmartLightInfo.IsOn)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_hue", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil || !report.Latest.SmartLightInfo.IsColour {
			return math.NaN()
		} else {
			return float64(report.Latest.SmartLightInfo.Hue)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_saturation", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil || !report.Latest.SmartLightInfo.IsColour {
			return math.NaN()
		} else {
			return float64(report.Latest.SmartLightInfo.Saturation)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_colour_temperature", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil || !report.Latest.SmartLightInfo.IsVariableColourTemperature {
			return math.NaN()
		} else {
			return float64(report.Latest.SmartLightInfo.ColourTemperature)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_brightness", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil {
			return math.NaN()
		} else {
			return float64(report.Latest.SmartLightInfo.Brightness)
		}
	}))

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_max_lumens", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil {
			return math.NaN()
		} else {
			return float64(report.Latest.SmartLightInfo.MaximumLumens)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_wattage", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.SmartLightInfo == nil {
			return math.NaN()
		} else {
			return float64(report.Latest.SmartLightInfo.Wattage)
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
			return math.NaN()
		} else {
			return float64(report.Latest.EnergyMeterInfo.CurrentMilliAmps)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_power_mw", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return math.NaN()
		} else {
			return float64(report.Latest.EnergyMeterInfo.PowerMilliWatts)
		}
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "kasa_voltage_mv", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return math.NaN()
		} else {
			return float64(report.Latest.EnergyMeterInfo.VoltageMilliVolts)
		}
	}))
	registry.MustRegister(prometheus.NewCounterFunc(prometheus.CounterOpts{Name: "kasa_totalEnergy_wh", ConstLabels: labels}, func() float64 {
		if report.Latest == nil || report.Latest.EnergyMeterInfo == nil {
			return math.NaN()
		} else {
			return float64(report.Latest.EnergyMeterInfo.TotalEnergyWattHours)
		}
	}))

}
