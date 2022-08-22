package device

import (
	"homepower/device/kasa"
	"homepower/types"
)

func ExtractAllData(device *types.DeviceConfig) {
	switch device.Model {
	case types.KasaHS100, types.KasaHS110, types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		kasa.ExtractAllData(device)
	default:
		panic("Unknown device model")
	}
}
