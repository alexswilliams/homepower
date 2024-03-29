package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"homepower/types"
	"log"
	"os"
)

func ReadConfigAndCredentials() *AppConfig {
	// for example:
	// HOMEPOWER_DEVICE_CONFIG_FILEPATH=config/exampleDeviceManifest.yaml
	// HOMEPOWER_CREDENTIAL_FILEPATH=config/exampleCredentials.yaml
	deviceConfigFilepath := os.Getenv("HOMEPOWER_DEVICE_CONFIG_FILEPATH")
	credentialFilepath := os.Getenv("HOMEPOWER_CREDENTIAL_FILEPATH")
	if deviceConfigFilepath == "" || credentialFilepath == "" {
		panic("environment variables for config file locations have not been set")
	}

	appConfig := &AppConfig{}
	readDeviceConfig(appConfig, deviceConfigFilepath)
	readCredentials(appConfig, credentialFilepath)
	log.Printf("Using device config: %+v\n", appConfig.Devices)
	return appConfig
}

func readDeviceConfig(appConfig *AppConfig, filepath string) {
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
	readConfig(filepath, &devicesFromYaml)
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

func readCredentials(config *AppConfig, filepath string) {
	type emailAndPassword struct {
		Email    string `yaml:"email"`
		Password string `yaml:"password"`
	}
	type credentialsFromFile struct {
		Tapo emailAndPassword `yaml:"tapo"`
	}
	credentials := credentialsFromFile{}
	readConfig(filepath, &credentials)
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
