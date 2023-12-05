package kasa

import "homepower/types"

func isSwitch(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS100 || config.Model == types.KasaHS110 || config.Model == types.KasaKP115
}

func isLight(config *types.DeviceConfig) bool {
	return config.Model == types.KasaKL50B || config.Model == types.KasaKL110B || config.Model == types.KasaKL130B
}

func isLightVariableTemperature(config *types.DeviceConfig) bool {
	return config.Model == types.KasaKL130B
}

func isLightColoured(config *types.DeviceConfig) bool {
	return config.Model == types.KasaKL130B
}

func hasPowerMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS110 || isLight(config) || config.Model == types.KasaKP115
}

func hasTotalEnergyMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS110 || config.Model == types.KasaKL50B || config.Model == types.KasaKL130B || config.Model == types.KasaKP115 // TODO: maybe not 130???
}

func hasCurrentAndVoltageMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS110 || config.Model == types.KasaKL50B || config.Model == types.KasaKP115 // TODO: really 50??
}

func supportsEMeter(config *types.DeviceConfig) bool {
	return hasPowerMonitoring(config) || hasTotalEnergyMonitoring(config) || hasCurrentAndVoltageMonitoring(config)
}
