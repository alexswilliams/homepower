package kasa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"homepower/types"
	"strconv"
	"strings"
	"time"
)

const sysInfoBody = `{"system":{"get_sysinfo":null}}`
const eMeterRealTimeQualifiedBody = `{"smartlife.iot.common.emeter":{"get_realtime":{}}}`
const eMeterRealTimeShortBody = `{"emeter":{"get_realtime":{}}}`
const lightingServiceLightDetailsBody = `{"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{}}}`

type Device struct {
	deviceConfig *types.DeviceConfig
	connection   *deviceConnection
	metrics      *prometheusMetrics
}

func NewDevice(config *types.DeviceConfig, registry prometheus.Registerer) *Device {
	connection := newDeviceConnection(config.Ip, 9999)
	return &Device{
		deviceConfig: config,
		connection:   connection,
		metrics:      registerMetrics(registry, config),
	}
}

func (dev *Device) PollDeviceAndUpdateMetrics() error {
	report, err := dev.extractAllData()
	if err != nil {
		return fmt.Errorf("could not poll device for info: %w", err)
	}
	if err := dev.metrics.updateMetrics(report); err != nil {
		return fmt.Errorf("could not update metrics after device poll: %w", err)
	}
	return nil
}
func (dev *Device) ResetMetricsToRogueValues() {
	dev.metrics.resetToRogueValues()
}
func (dev *Device) ResetDeviceConnection() {
	dev.connection.closeCurrentConnection()
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

func eMeterQueryForDevice(model types.DeviceType) (queryBody string) {
	switch model {
	case types.KasaHS110:
		return eMeterRealTimeShortBody
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		return eMeterRealTimeQualifiedBody
	default:
		panic("Device has invalid model type for the Kasa driver")
	}
}

func (dev *Device) extractAllData() (*periodicDeviceReport, error) {
	var startTime = time.Now()
	err := dev.connection.openNewConnection()
	defer dev.connection.closeCurrentConnection()
	if err != nil {
		return nil, fmt.Errorf("could not create connection when extracting data: %w", err)
	}

	var deviceInfoJson []byte
	if deviceInfoJson, err = dev.connection.queryDevice(sysInfoBody); err != nil {
		return nil, fmt.Errorf("could not query for device info: %w", err)
	}

	var realTimeJson []byte
	if supportsEMeter(dev.deviceConfig) {
		var eMeterRealTimeBody = eMeterQueryForDevice(dev.deviceConfig.Model)
		if realTimeJson, err = dev.connection.queryDevice(eMeterRealTimeBody); err != nil {
			return nil, fmt.Errorf("could not query for eMeter info: %w", err)
		}
	}

	var lampInfoJson []byte
	if isLight(dev.deviceConfig) {
		if lampInfoJson, err = dev.connection.queryDevice(lightingServiceLightDetailsBody); err != nil {
			return nil, fmt.Errorf("could not query for lamp info: %w", err)
		}
	}

	return buildPeriodicDeviceReport(dev.deviceConfig.Model, deviceInfoJson,
		lampInfoJson, isLight(dev.deviceConfig), realTimeJson, supportsEMeter(dev.deviceConfig), startTime)
}

func buildPeriodicDeviceReport(
	model types.DeviceType,
	deviceInfo []byte,
	lampInfo []byte, supportsLampInfo bool,
	realTime []byte, supportsEMeter bool,
	startTime time.Time) (*periodicDeviceReport, error) {

	var report = periodicDeviceReport{}

	if err := appendDeviceInfo(model, deviceInfo, &report); err != nil {
		return &report, fmt.Errorf("could not merge device info into report: %w", err)
	}
	if supportsEMeter {
		if err := appendEMeterInfo(model, realTime, &report); err != nil {
			return &report, fmt.Errorf("could not merge eMeter info into report: %w", err)
		}
	}
	if supportsLampInfo {
		if err := appendLampInfo(lampInfo, &report); err != nil {
			return &report, fmt.Errorf("could not merge lamp info into report: %w", err)
		}
	}
	report.ScrapeDuration = time.Since(startTime)
	return &report, nil
}

func appendDeviceInfo(model types.DeviceType, deviceInfo []byte, report *periodicDeviceReport) error {
	var infoJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(deviceInfo, &infoJson); err != nil {
		return fmt.Errorf("could not unmarshal device info json: %w", err)
	}
	var data = infoJson["system"]["get_sysinfo"]
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch system info returned non-zero err_code: " + strconv.Itoa(int(data["err_code"].(float64))))
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

func appendLampInfo(lampInfo []byte, report *periodicDeviceReport) error {
	var lampJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(lampInfo, &lampJson); err != nil {
		return fmt.Errorf("could not unmarshal lamp info json: %w", err)
	}
	var data = lampJson["smartlife.iot.smartbulb.lightingservice"]["get_light_details"]
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch lamp info data returned non-zero err_code: " + strconv.Itoa(int(data["err_code"].(float64))))
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
		return fmt.Errorf("could not unmarshal eMeter info json: %w", err)
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
		return errors.New("call to fetch eMeter data returned non-zero err_code: " + strconv.Itoa(int(data["err_code"].(float64))))
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
