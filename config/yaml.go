package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"homepower/types"
	"log"
	"os"
)

func ReadConfigAndCredentials() *AppConfig {
	appConfig := &AppConfig{}
	readDeviceConfig(appConfig)
	readCredentials(appConfig)
	log.Printf("Using device config: %+v\n", appConfig.Devices)
	return appConfig
}

func readDeviceConfig(appConfig *AppConfig) {
	type deviceFromFile struct {
		Name   string `yaml:"name"`
		Room   string `yaml:"room"`
		Ip     string `yaml:"ip"`
		Model  string `yaml:"model"`
		Driver string `yaml:"driver"`
	}
	type devicesConfigFile struct {
		Devices []deviceFromFile `yaml:"devices"`
	}
	devicesFromYaml := devicesConfigFile{}
	readConfig("config/exampleDeviceManifest.yaml", &devicesFromYaml)
	appConfig.Devices = make([]types.DeviceConfig, 0, len(devicesFromYaml.Devices))
	for _, device := range devicesFromYaml.Devices {
		appConfig.Devices = append(appConfig.Devices, types.DeviceConfig{
			Name:  device.Name,
			Room:  device.Room,
			Model: types.DeviceTypeFor(device.Model),
			Ip:    device.Ip,
		})
	}
}

func readCredentials(config *AppConfig) {
	type emailAndPassword struct {
		Email    string `yaml:"email"`
		Password string `yaml:"password"`
	}
	type credentialsFromFile struct {
		Tapo emailAndPassword `yaml:"tapo"`
	}
	credentials := credentialsFromFile{}
	readConfig("config/exampleCredentials.yaml", &credentials)
	config.TapoCredentials.EmailAddress = credentials.Tapo.Email
	config.TapoCredentials.Password = credentials.Tapo.Password
}

func readConfig[E any](filename string, into *E) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Errorf("could not read config file '%s': %w", filename, err))
	}
	err = yaml.Unmarshal(fileBytes, into)
	if err != nil {
		panic(fmt.Errorf("could not unmarshal config file yaml '%s': %w", filename, err))
	}
}
