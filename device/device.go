package device

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/config"
	"homepower/device/kasa"
	"homepower/device/tapo_klap"
	"homepower/types"
)

func Factory(deviceConfig types.DeviceConfig, tapoCredentials *config.Credentials, registry prometheus.Registerer) (types.PollableDevice, error) {
	switch types.DriverFor(deviceConfig.Model) {
	case types.Kasa:
		return kasa.NewDevice(&deviceConfig, registry), nil
	case types.Tapo:
		return tapo_klap.NewDevice(tapoCredentials.EmailAddress, tapoCredentials.Password, &deviceConfig, registry, 80)
	//case types.Tapo:
	//	return tapo.NewDevice(tapoCredentials.EmailAddress, tapoCredentials.Password, &deviceConfig, registry, 80)
	default:
		return nil, errors.New("unknown device type")
	}
}
