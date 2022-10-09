package config

import (
	"homepower/types"
)

type AppConfig struct {
	Devices         []types.DeviceConfig
	TapoCredentials Credentials
}

type Credentials struct {
	EmailAddress string
	Password     string
}
