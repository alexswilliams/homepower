package tapo

import (
	"encoding/base64"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"strings"
	"time"
)

type Device struct {
	deviceConfig *types.DeviceConfig
	connection   *deviceConnection
	metrics      prometheusMetrics
}

func NewDevice(email string, password string, config *types.DeviceConfig, registry prometheus.Registerer) (*Device, error) {
	connection, err := newDeviceConnection(email, password, config.Ip, 80)
	if err != nil {
		return nil, fmt.Errorf("could not initialise connection for device %s (%s): %w", config.Ip, config.Name, err)
	}
	return &Device{
		deviceConfig: config,
		connection:   connection,
		metrics: registerMetrics(
			registry,
			types.GenerateCommonLabels(config),
			isSwitch(config), isLight(config), hasEnergyMonitoring(config)),
	}, nil
}

type DeviceStatus struct {
	Common
	*SmartPlugInfo
	*SmartBulbInfo
	*EnergyMeterInfo
}

type Common struct {
	Alias           string
	DeviceId        string
	FirmwareVersion string
	HardwareId      string
	Mac             string
	ModelName       string
	OemId           string
	Overheated      bool
	WifiRssi        int
	SignalLevel     int
	DeviceType      string // e.g. SMART.TAPOBULB, SMART.TAPOPLUG
}
type SmartPlugInfo struct {
	RelayOn bool
	OnTime  time.Duration
}
type SmartBulbInfo struct {
	Brightness        int
	ColourTemperature int
	LightOn           bool
	Hue               int
	Saturation        int
}
type EnergyMeterInfo struct {
	PowerMilliWatts      int
	MonthEnergyWattHours int
	TodayEnergyWattHours int
}

func (dev *Device) PopulateDeviceInfo(status *DeviceStatus) error {
	responseResult, err := dev.connection.makeApiCall("get_device_info", nil)
	if err != nil {
		return fmt.Errorf("could not make API call while fetching device info: %w", err)
	}
	// log.Println(responseResult)

	nicknameBase64 := responseResult["nickname"].(string)
	if alias, err := base64.StdEncoding.DecodeString(nicknameBase64); err == nil {
		status.Alias = strings.TrimSpace(string(alias))
	} else {
		status.Alias = nicknameBase64
	}
	status.DeviceId = responseResult["device_id"].(string)
	status.FirmwareVersion = responseResult["fw_ver"].(string)
	status.HardwareId = responseResult["hw_id"].(string)
	status.Mac = strings.ReplaceAll(responseResult["mac"].(string), "-", "")
	status.ModelName = responseResult["model"].(string)
	status.OemId = responseResult["oem_id"].(string)
	status.Overheated = responseResult["overheated"].(bool)
	status.WifiRssi = int(responseResult["rssi"].(float64))
	status.SignalLevel = int(responseResult["signal_level"].(float64))
	status.DeviceType = responseResult["type"].(string)

	if status.DeviceType == "SMART.TAPOBULB" {
		status.SmartBulbInfo = &SmartBulbInfo{
			Brightness:        int(responseResult["brightness"].(float64)),
			ColourTemperature: int(responseResult["color_temp"].(float64)),
			LightOn:           responseResult["device_on"].(bool),
			Hue:               int(responseResult["hue"].(float64)),
			Saturation:        int(responseResult["saturation"].(float64)),
		}
	} else if status.DeviceType == "SMART.TAPOPLUG" {
		status.SmartPlugInfo = &SmartPlugInfo{
			RelayOn: responseResult["device_on"].(bool),
			OnTime:  time.Duration(int64(responseResult["on_time"].(float64))) * time.Second,
		}
	}
	return nil
}

func (dev *Device) PopulateEnergyInfo(status *DeviceStatus) error {
	responseResult, err := dev.connection.makeApiCall("get_energy_usage", nil)
	if err != nil {
		return fmt.Errorf("could not make API call while fetching energy usage: %w", err)
	}
	// log.Println(responseResult)
	status.EnergyMeterInfo = &EnergyMeterInfo{
		PowerMilliWatts:      int(responseResult["current_power"].(float64)),
		MonthEnergyWattHours: int(responseResult["month_energy"].(float64)),
		TodayEnergyWattHours: int(responseResult["today_energy"].(float64)),
	}
	return nil
}

func (dev *Device) UpdateMetrics() error {
	var status = DeviceStatus{}
	if err := dev.PopulateDeviceInfo(&status); err != nil {
		return fmt.Errorf("could not poll device info for %s (%s): %w", dev.deviceConfig.Ip, dev.deviceConfig.Name, err)
	}
	if hasEnergyMonitoring(dev.deviceConfig) {
		if err := dev.PopulateEnergyInfo(&status); err != nil {
			return fmt.Errorf("could not poll energy info for %s (%s): %w", dev.deviceConfig.Ip, dev.deviceConfig.Name, err)
		}
	}
	if err := dev.metrics.updateMetrics(&status); err != nil {
		return fmt.Errorf("could not update metrics for %s (%s): %w", dev.deviceConfig.Ip, dev.deviceConfig.Name, err)
	}
	//log.Printf("%+v", status)
	//log.Printf("%+v", status.SmartBulbInfo)
	//log.Printf("%+v", status.SmartPlugInfo)
	//log.Printf("%+v", status.EnergyMeterInfo)
	return nil
}

func isSwitch(config *types.DeviceConfig) bool {
	return config.Model == types.TapoP100 || config.Model == types.TapoP110
}

func isLight(config *types.DeviceConfig) bool {
	return config.Model == types.TapoL900
}

func hasEnergyMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.TapoP110
}
