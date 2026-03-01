package device

import (
	"errors"
	"homepower/config"
	"homepower/device/kasa"
	"homepower/device/tapo"
	"homepower/types"

	"github.com/prometheus/client_golang/prometheus"
)

func Factory(deviceConfig types.DeviceConfig, tapoCredentials *config.Credentials, registry prometheus.Registerer) (types.PollableDevice, error) {
	switch types.DriverFor(deviceConfig.Model) {
	case types.Kasa:
		return kasa.NewDevice(&deviceConfig, registry), nil
	case types.Tapo:
		return tapo.NewDevice(tapoCredentials.EmailAddress, tapoCredentials.Password, &deviceConfig, registry, 80)
	default:
		return nil, errors.New("unknown device type")
	}
}
