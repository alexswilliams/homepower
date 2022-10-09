package kasa

import (
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"strings"
	"time"
)

const sysInfoBody = `{"system":{"get_sysinfo":null}}`
const eMeterRealTimeQualifiedBody = `{"smartlife.iot.common.emeter":{"get_realtime":{}}}`
const eMeterRealTimeShortBody = `{"emeter":{"get_realtime":{}}}`
const lightingServiceLightDetailsBody = `{"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{}}}`

type Device struct {
	deviceConfig *types.DeviceConfig
	connection   any
	metrics      prometheusMetrics
}

func NewDevice(config *types.DeviceConfig, registry prometheus.Registerer) (*Device, error) {
	return &Device{
		deviceConfig: config,
		metrics:      registerMetrics(registry, config),
	}, nil
}

func (dev *Device) PollDeviceAndUpdateMetrics() error {
	report, err := extractAllData(dev.deviceConfig)
	if err != nil {
		return err
	}
	if err := dev.metrics.updateMetrics(report); err != nil {
		return err
	}
	return nil
}
func (dev *Device) ResetMetricsToRogueValues() {
	dev.metrics.resetToRogueValues()
}
func (dev *Device) CommonMetricLabels() map[string]string {
	return types.GenerateCommonLabels(dev.deviceConfig)
}

type periodicDeviceReport struct {
	common
	*smartBulbInfo
	*smartPlugInfo
	*energyMeterInfo
}

type common struct {
	ActiveMode       string
	Alias            string
	ModelDescription string
	DeviceId         string
	SoftwareVersion  string
	HardwareId       string
	OemId            string
	Mac              string
	ModelName        string
	WifiRssi         int
	DeviceType       string
	ScrapeDuration   time.Duration
}

type smartPlugInfo struct {
	RelayOn  bool
	LedOn    bool
	OnTime   time.Duration
	Updating bool
}

type smartBulbInfo struct {
	DeviceState                 string
	IsOn                        bool
	IsDimmable                  bool
	IsColour                    bool
	IsVariableColourTemperature bool
	Mode                        string
	Hue                         int
	Saturation                  int
	ColourTemperature           int
	Brightness                  int
	LampBeamAngle               int
	MinimumVoltage              int
	MaximumVoltage              int
	Wattage                     int
	IncandescentEquivalent      int
	MaximumLumens               int
}

type energyMeterInfo struct {
	VoltageMilliVolts    int
	CurrentMilliAmps     int
	PowerMilliWatts      int
	TotalEnergyWattHours int
}

func eMeterQueryForDevice(model types.DeviceType) (queryBody string, supportsEMeter bool) {
	switch model {
	case types.KasaHS100:
		return "", false
	case types.KasaHS110:
		return eMeterRealTimeShortBody, true
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return eMeterRealTimeQualifiedBody, true
	default:
		panic("Device has invalid model type for the Kasa driver")
	}
}

func lightDetailsQueryForDevice(model types.DeviceType) (queryBody string, supportsLampInfo bool) {
	switch model {
	case types.KasaHS100, types.KasaHS110:
		return "", false
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return lightingServiceLightDetailsBody, true
	default:
		panic("Device has invalid model type for the Kasa driver")
	}
}

func extractAllData(device *types.DeviceConfig) (*periodicDeviceReport, error) {
	var startTime = time.Now()
	connection, err := openConnection(device.Ip, 9999)
	if err != nil {
		return nil, err
	}
	defer func() { _ = connection.Close() }()

	// All kasa devices support the `get_sysinfo` query
	var deviceInfoJson []byte
	if deviceInfoJson, err = queryDevice(connection, sysInfoBody); err != nil {
		return nil, err
	}

	var eMeterRealTimeBody, supportsEMeter = eMeterQueryForDevice(device.Model)
	var realTimeJson []byte
	if supportsEMeter {
		if realTimeJson, err = queryDevice(connection, eMeterRealTimeBody); err != nil {
			return nil, err
		}
	}

	var lampInfoQueryBody, supportsLampInfo = lightDetailsQueryForDevice(device.Model)
	var lampInfoJson []byte
	if supportsLampInfo {
		if lampInfoJson, err = queryDevice(connection, lampInfoQueryBody); err != nil {
			return nil, err
		}

	}

	return buildPeriodicDeviceReport(device.Model, deviceInfoJson,
		lampInfoJson, supportsLampInfo, realTimeJson, supportsEMeter, startTime)
}

func buildPeriodicDeviceReport(
	model types.DeviceType,
	deviceInfo []byte,
	lampInfo []byte, supportsLampInfo bool,
	realTime []byte, supportsEMeter bool,
	startTime time.Time) (*periodicDeviceReport, error) {

	var report = periodicDeviceReport{}

	err := appendDeviceInfo(model, deviceInfo, &report)
	if err != nil {
		return &report, err
	}

	if supportsEMeter {
		err2 := appendEMeterInfo(model, realTime, &report)
		if err2 != nil {
			return &report, err2
		}
	}

	if supportsLampInfo {
		err2 := appendLampInfo(lampInfo, &report)
		if err2 != nil {
			return &report, err2
		}
	}
	report.ScrapeDuration = time.Since(startTime)
	return &report, nil
}

// region Device Info
func appendDeviceInfo(model types.DeviceType, deviceInfo []byte, report *periodicDeviceReport) error {
	var infoJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(deviceInfo, &infoJson); err != nil {
		return err
	}
	var data = infoJson["system"]["get_sysinfo"]
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch system info failed")
	}

	report.common = common{
		ActiveMode:       data["active_mode"].(string),     // e.g. "none"
		Alias:            data["alias"].(string),           // e.g. "Living Room Ceiling Light"
		ModelDescription: mapModelDescription(model, data), // e.g. Smart Wi-Fi LED Bulb with Dimmable Light
		DeviceId:         data["deviceId"].(string),        // e.g. AABB0011CC22DD33EE44FF550011CC33FF55AA77
		HardwareId:       data["hwId"].(string),            // e.g. 00112233445566778899AA00BB00CC00
		SoftwareVersion:  data["sw_ver"].(string),          // e.g. 1.5.10 Build 191125 Rel.094314
		OemId:            data["oemId"].(string),           // e.g. E57A51C2293DD01A3171CD7949972746
		Mac:              mapMac(model, data),              // e.g. AA0011BB2233
		ModelName:        data["model"].(string),           // e.g. KL50B(UN)
		WifiRssi:         int(data["rssi"].(float64)),      // e.g. -58
		DeviceType:       mapDeviceType(model, data),       // IOT.SMARTPLUGSWITCH or IOT.SMARTBULB
	}

	switch model {
	case types.KasaHS100, types.KasaHS110:
		report.smartPlugInfo = &smartPlugInfo{
			RelayOn:  int(data["relay_state"].(float64)) == 1,
			LedOn:    int(data["led_off"].(float64)) == 0,
			OnTime:   time.Duration(int64(data["on_time"].(float64))) * time.Second,
			Updating: int(data["updating"].(float64)) != 0,
		}
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		report.smartBulbInfo = &smartBulbInfo{
			DeviceState:                 data["dev_state"].(string), // e.g. "normal"
			IsDimmable:                  int(data["is_dimmable"].(float64)) == 1,
			IsOn:                        false,
			IsColour:                    int(data["is_color"].(float64)) == 1,
			IsVariableColourTemperature: int(data["is_variable_color_temp"].(float64)) == 1,
		}
		var lightState = data["light_state"].(map[string]interface{})
		report.smartBulbInfo.IsOn = int(lightState["on_off"].(float64)) == 1
		if report.smartBulbInfo.IsOn {
			report.smartBulbInfo.Mode = lightState["mode"].(string)
			report.smartBulbInfo.Hue = int(lightState["hue"].(float64))
			report.smartBulbInfo.Saturation = int(lightState["saturation"].(float64))
			report.smartBulbInfo.ColourTemperature = int(lightState["color_temp"].(float64))
			report.smartBulbInfo.Brightness = int(lightState["brightness"].(float64))
		}
	}
	return nil
}

func mapDeviceType(model types.DeviceType, data map[string]interface{}) string {
	switch model {
	case types.KasaHS100, types.KasaHS110:
		return data["type"].(string)
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return data["mic_type"].(string)
	default:
		panic("Device has invalid model type for the Kasa driver")
	}
}

func mapMac(model types.DeviceType, data map[string]interface{}) string {
	switch model {
	case types.KasaHS100, types.KasaHS110: // e.g. AA:00:11:BB:22:33
		return strings.ReplaceAll(data["mac"].(string), ":", "")
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B: // e.g. AA0011BB2233
		return data["mic_mac"].(string)
	default:
		panic("Device has invalid model type for the Kasa driver")
	}
}

func mapModelDescription(model types.DeviceType, data map[string]interface{}) string {
	switch model {
	case types.KasaHS100, types.KasaHS110:
		return data["dev_name"].(string)
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return data["description"].(string)
	default:
		panic("Device has invalid model type for the Kasa driver")
	}
}

//endregion

func appendLampInfo(lampInfo []byte, report *periodicDeviceReport) error {
	var lampJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(lampInfo, &lampJson); err != nil {
		return err
	}
	var data = lampJson["smartlife.iot.smartbulb.lightingservice"]["get_light_details"]
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch lamp info data failed")
	}
	if report.smartBulbInfo == nil {
		report.smartBulbInfo = &smartBulbInfo{}
	}
	report.smartBulbInfo.LampBeamAngle = int(data["lamp_beam_angle"].(float64))
	report.smartBulbInfo.MinimumVoltage = int(data["min_voltage"].(float64))
	report.smartBulbInfo.MaximumVoltage = int(data["max_voltage"].(float64))
	report.smartBulbInfo.Wattage = int(data["wattage"].(float64))
	report.smartBulbInfo.IncandescentEquivalent = int(data["incandescent_equivalent"].(float64))
	report.smartBulbInfo.MaximumLumens = int(data["max_lumens"].(float64))
	return nil
}

func appendEMeterInfo(model types.DeviceType, realTime []byte, report *periodicDeviceReport) error {
	var eMeterJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(realTime, &eMeterJson); err != nil {
		return err
	}
	var data map[string]interface{}
	switch model {
	case types.KasaHS110:
		data = eMeterJson["emeter"]["get_realtime"]
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		data = eMeterJson["smartlife.iot.common.emeter"]["get_realtime"]
	default:
		panic("Device has invalid model type for the Kasa eMeter driver")
	}
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch eMeter data failed")
	}

	report.energyMeterInfo = &energyMeterInfo{
		PowerMilliWatts: int(data["power_mw"].(float64)),
	}
	if voltage, prs := data["voltage_mv"]; prs {
		report.energyMeterInfo.VoltageMilliVolts = int(voltage.(float64))
	}
	if current, prs := data["current_ma"]; prs {
		report.energyMeterInfo.CurrentMilliAmps = int(current.(float64))
	}
	if totalEnergy, prs := data["total_wh"]; prs {
		report.energyMeterInfo.TotalEnergyWattHours = int(totalEnergy.(float64))
	}
	return nil
}

func isSwitch(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS100 || config.Model == types.KasaHS110
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
	return config.Model == types.KasaHS110 || isLight(config)
}

func hasTotalEnergyMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS110 || config.Model == types.KasaKL50B || config.Model == types.KasaKL130B // TODO: maybe not 130???
}

func hasCurrentAndVoltageMonitoring(config *types.DeviceConfig) bool {
	return config.Model == types.KasaHS110 || config.Model == types.KasaKL50B // TODO: really 50??
}
