package tapo_klap

import "homepower/types"

func isSwitch(config *types.DeviceConfig) bool {
	return config.Model == types.TapoP100 || config.Model == types.TapoP110
}

func isLight(config *types.DeviceConfig) bool {
	return config.Model == types.TapoL900
}

func hasEnergyMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.TapoP110
}
