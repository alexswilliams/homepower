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
	connection   tapoDeviceConnection
	metrics      *prometheusMetrics
}

func NewDevice(email string, password string, config *types.DeviceConfig, registry prometheus.Registerer, port uint16) (*Device, error) {
	var connection tapoDeviceConnection = connectionFactory(email, password, config.Ip, port)
	return &Device{
		deviceConfig: config,
		connection:   connection,
		metrics: registerMetrics(
			registry,
			types.GenerateCommonLabels(config),
			isSwitch(config), isLight(config), hasEnergyMonitoring(config)),
	}, nil
}

func (dev *Device) PollDeviceAndUpdateMetrics() error {
	var status = deviceStatus{}
	if err := dev.populateDeviceInfo(&status); err != nil {
		return fmt.Errorf("could not poll device info for %s (%s): %w", dev.deviceConfig.Ip, dev.deviceConfig.Name, err)
	}
	if hasEnergyMonitoring(dev.deviceConfig) {
		if err := dev.populateEnergyInfo(&status); err != nil {
			return fmt.Errorf("could not poll energy info for %s (%s): %w", dev.deviceConfig.Ip, dev.deviceConfig.Name, err)
		}
	}
	if err := dev.metrics.updateMetrics(&status); err != nil {
		return fmt.Errorf("could not update metrics for %s (%s): %w", dev.deviceConfig.Ip, dev.deviceConfig.Name, err)
	}
	return nil
}
func (dev *Device) ResetMetricsToRogueValues() {
	dev.metrics.resetToRogueValues()
}
func (dev *Device) ResetDeviceConnection() {
	dev.connection.forgetKeysAndSession()
}
func (dev *Device) CommonMetricLabels() map[string]string {
	return dev.metrics.commonLabels
}

type deviceStatus struct {
	common
	*smartPlugInfo
	*smartBulbInfo
	*energyMeterInfo
}

type common struct {
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
type smartPlugInfo struct {
	RelayOn bool
	OnTime  time.Duration
}
type smartBulbInfo struct {
	Brightness        int
	ColourTemperature int
	LightOn           bool
	Hue               int
	Saturation        int
}
type energyMeterInfo struct {
	PowerMilliWatts      int
	MonthEnergyWattHours int
	TodayEnergyWattHours int
}

func (dev *Device) populateDeviceInfo(status *deviceStatus) error {
	responseResult, err := dev.connection.GetDeviceInfo()
	if err != nil {
		return fmt.Errorf("could not make API call while fetching device info: %w", err)
	}

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
		status.smartBulbInfo = &smartBulbInfo{
			Brightness:        int(responseResult["brightness"].(float64)),
			ColourTemperature: int(responseResult["color_temp"].(float64)),
			LightOn:           responseResult["device_on"].(bool),
			Hue:               int(responseResult["hue"].(float64)),
			Saturation:        int(responseResult["saturation"].(float64)),
		}
	} else if status.DeviceType == "SMART.TAPOPLUG" {
		status.smartPlugInfo = &smartPlugInfo{
			RelayOn: responseResult["device_on"].(bool),
			OnTime:  time.Duration(int64(responseResult["on_time"].(float64))) * time.Second,
		}
	}
	return nil
}

func (dev *Device) populateEnergyInfo(status *deviceStatus) error {
	responseResult, err := dev.connection.GetEnergyUsage()
	if err != nil {
		return fmt.Errorf("could not make API call while fetching energy usage: %w", err)
	}
	status.energyMeterInfo = &energyMeterInfo{
		PowerMilliWatts:      int(responseResult["current_power"].(float64)),
		MonthEnergyWattHours: int(responseResult["month_energy"].(float64)),
		TodayEnergyWattHours: int(responseResult["today_energy"].(float64)),
	}
	return nil
}
