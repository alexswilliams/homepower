package device

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/device/kasa"
	"homepower/types"
)

func ExtractAllData(device *types.DeviceConfig) (*kasa.PeriodicDeviceReport, error) {
	switch device.Model {
	case types.KasaHS100, types.KasaHS110, types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return kasa.ExtractAllData(device)
	default:
		return nil, errors.New("unknown device type")
	}
}

func RegisterMetrics(registry prometheus.Registerer, dev types.DeviceConfig, lastDeviceReport *kasa.LatestDeviceReport) *prometheus.GaugeVec {
	switch dev.Model {
	case types.KasaHS100, types.KasaHS110, types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return kasa.RegisterMetrics(registry, dev, lastDeviceReport)
	}
	return nil
}
