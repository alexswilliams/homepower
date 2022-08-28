package kasa

import (
	"encoding/json"
	"errors"
	"homepower/types"
	"strings"
	"time"
)

const sysInfoBody = `{"system":{"get_sysinfo":null}}`
const eMeterRealTimeQualifiedBody = `{"smartlife.iot.common.emeter":{"get_realtime":{}}}`
const eMeterRealTimeShortBody = `{"emeter":{"get_realtime":{}}}`
const lightingServiceLightDetailsBody = `{"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{}}}`

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

func ExtractAllData(device *types.DeviceConfig) (*PeriodicDeviceReport, error) {
	var startTime = time.Now()
	connection, err := openConnection(device.Ip, 9999)
	if err != nil {
		return nil, err
	}
	defer func() { _ = connection.Close() }()

	// All kasa devices support the `get_sysinfo` query
	var deviceInfoJson = queryDevice(connection, sysInfoBody)

	var eMeterRealTimeBody, supportsEMeter = eMeterQueryForDevice(device.Model)
	var realTimeJson []byte
	if supportsEMeter {
		realTimeJson = queryDevice(connection, eMeterRealTimeBody)
	}

	var lampInfoQueryBody, supportsLampInfo = lightDetailsQueryForDevice(device.Model)
	var lampInfoJson []byte
	if supportsLampInfo {
		lampInfoJson = queryDevice(connection, lampInfoQueryBody)
	}

	report, err := buildPeriodicDeviceReport(device.Model, deviceInfoJson,
		lampInfoJson, supportsLampInfo, realTimeJson, supportsEMeter, startTime)
	return report, err
}

func buildPeriodicDeviceReport(
	model types.DeviceType,
	deviceInfo []byte,
	lampInfo []byte, supportsLampInfo bool,
	realTime []byte, supportsEMeter bool,
	startTime time.Time) (*PeriodicDeviceReport, error) {

	var report = PeriodicDeviceReport{}

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
func appendDeviceInfo(model types.DeviceType, deviceInfo []byte, report *PeriodicDeviceReport) error {
	// HS100: {"system":{"get_sysinfo":{"sw_ver":"1.5.10 Build 191125 Rel.094314","hw_ver":"2.1","type":"IOT.SMARTPLUGSWITCH","model":"HS100(UK)","mac":"68:FF:7B:A6:12:5E","dev_name":"Smart Wi-Fi Plug","alias":"Christmas Lights","relay_state":0,"on_time":0,"active_mode":"none","feature":"TIM","updating":0,"icon_hash":"","rssi":-39,"led_off":0,"longitude_i":-15663,"latitude_i":538250,"hwId":"82589DCE59161C80EC57E0A2834D25A2","fwId":"00000000000000000000000000000000","deviceId":"8006F4838363F8F93D29E965F44ECB6F1B7148D2","oemId":"FDD18403D5E8DB3613009C820963E018","next_action":{"type":-1},"ntc_state":0,"err_code":0}}}
	// HS110: {"system":{"get_sysinfo":{"sw_ver":"1.5.10 Build 191125 Rel.094314","hw_ver":"2.1","type":"IOT.SMARTPLUGSWITCH","model":"HS110(UK)","mac":"D8:0D:17:6C:7D:47","dev_name":"Smart Wi-Fi Plug With Energy Monitoring","alias":"Work Desk","relay_state":1,"on_time":20072,"active_mode":"none","feature":"TIM:ENE","updating":0,"icon_hash":"","rssi":-51,"led_off":0,"longitude_i":-15663,"latitude_i":538250,"hwId":"0750E2C15BB77902833ABF45366B8E9A","fwId":"00000000000000000000000000000000","deviceId":"80063164343B2A1209DE9D3067ADE9D21B0B5A43","oemId":"AB8C79FE7869756511CDC455BDFE41EA","next_action":{"type":-1},"ntc_state":0,"err_code":0}}}
	// KL50B Off: {"system":{"get_sysinfo":{"sw_ver":"1.1.13 Build 210524 Rel.082619","hw_ver":"1.0","model":"KL50B(UN)","deviceId":"80121831F908BEC919BCAE48D1C5BF461CF0F103","oemId":"E57A51C2293DD01A3171CD7949972746","hwId":"761989C14891B40717CC25C84FAAB1EE","rssi":-50,"longitude_i":-15663,"latitude_i":538250,"alias":"Office Ceiling Light","status":"new","description":"Kasa Smart Edison Bulb, Dimmable","mic_type":"IOT.SMARTBULB","mic_mac":"D847321B5F63","dev_state":"normal","is_factory":false,"disco_ver":"1.0","ctrl_protocols":{"name":"Linkie","version":"1.0"},"active_mode":"none","is_dimmable":1,"is_color":0,"is_variable_color_temp":0,"light_state":{"on_off":0,"dft_on_state":{"mode":"normal","hue":0,"saturation":0,"color_temp":2700,"brightness":10}},"preferred_state":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":100},{"index":1,"hue":0,"saturation":0,"color_temp":2700,"brightness":75},{"index":2,"hue":0,"saturation":0,"color_temp":2700,"brightness":25},{"index":3,"hue":0,"saturation":0,"color_temp":2700,"brightness":1}],"err_code":0}}}
	// KL50B On: {"system":{"get_sysinfo":{"sw_ver":"1.1.13 Build 210524 Rel.082619","hw_ver":"1.0","model":"KL50B(UN)","deviceId":"80121831F908BEC919BCAE48D1C5BF461CF0F103","oemId":"E57A51C2293DD01A3171CD7949972746","hwId":"761989C14891B40717CC25C84FAAB1EE","rssi":-46,"longitude_i":-15663,"latitude_i":538250,"alias":"Office Ceiling Light","status":"new","description":"Kasa Smart Edison Bulb, Dimmable","mic_type":"IOT.SMARTBULB","mic_mac":"D847321B5F63","dev_state":"normal","is_factory":false,"disco_ver":"1.0","ctrl_protocols":{"name":"Linkie","version":"1.0"},"active_mode":"none","is_dimmable":1,"is_color":0,"is_variable_color_temp":0,"light_state":{"on_off":1,"mode":"normal","hue":0,"saturation":0,"color_temp":2700,"brightness":100},"preferred_state":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":100},{"index":1,"hue":0,"saturation":0,"color_temp":2700,"brightness":75},{"index":2,"hue":0,"saturation":0,"color_temp":2700,"brightness":25},{"index":3,"hue":0,"saturation":0,"color_temp":2700,"brightness":1}],"err_code":0}}}
	// KL110B Off: {"system":{"get_sysinfo":{"sw_ver":"1.8.11 Build 191113 Rel.105336","hw_ver":"1.0","model":"KL110B(UN)","description":"Smart Wi-Fi LED Bulb with Dimmable Light","alias":"Hallway Ceiling Light","mic_type":"IOT.SMARTBULB","dev_state":"normal","mic_mac":"68FF7B4393F8","deviceId":"80125A77AB43254163ADF0BA914D829C1B4F3780","oemId":"50D44BFB1B9DE7D6E7E1B848B907CCDA","hwId":"111E35908497A05512E259BB76801E10","is_factory":false,"disco_ver":"1.0","ctrl_protocols":{"name":"Linkie","version":"1.0"},"light_state":{"on_off":0,"dft_on_state":{"mode":"normal","hue":0,"saturation":0,"color_temp":2700,"brightness":25}},"is_dimmable":1,"is_color":0,"is_variable_color_temp":0,"preferred_state":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":100},{"index":1,"hue":0,"saturation":0,"color_temp":2700,"brightness":75},{"index":2,"hue":0,"saturation":0,"color_temp":2700,"brightness":25},{"index":3,"hue":0,"saturation":0,"color_temp":2700,"brightness":1}],"rssi":-51,"active_mode":"none","heapsize":290628,"err_code":0}}}
	// KL110B On: {"system":{"get_sysinfo":{"sw_ver":"1.8.11 Build 191113 Rel.105336","hw_ver":"1.0","model":"KL110B(UN)","description":"Smart Wi-Fi LED Bulb with Dimmable Light","alias":"Hallway Ceiling Light","mic_type":"IOT.SMARTBULB","dev_state":"normal","mic_mac":"68FF7B4393F8","deviceId":"80125A77AB43254163ADF0BA914D829C1B4F3780","oemId":"50D44BFB1B9DE7D6E7E1B848B907CCDA","hwId":"111E35908497A05512E259BB76801E10","is_factory":false,"disco_ver":"1.0","ctrl_protocols":{"name":"Linkie","version":"1.0"},"light_state":{"on_off":1,"mode":"normal","hue":0,"saturation":0,"color_temp":2700,"brightness":25},"is_dimmable":1,"is_color":0,"is_variable_color_temp":0,"preferred_state":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":100},{"index":1,"hue":0,"saturation":0,"color_temp":2700,"brightness":75},{"index":2,"hue":0,"saturation":0,"color_temp":2700,"brightness":25},{"index":3,"hue":0,"saturation":0,"color_temp":2700,"brightness":1}],"rssi":-50,"active_mode":"none","heapsize":293416,"err_code":0}}}
	// KL130B Off: {"system":{"get_sysinfo":{"sw_ver":"1.0.12 Build 210329 Rel.141126","hw_ver":"2.0","model":"KL130B(UN)","deviceId":"801211B9312B531B26C449346D30572D1DCE005F","oemId":"E45F76AD3AF13E60B58D6F68739CD7E4","hwId":"1E97141B9F0E939BD8F9679F0B6167C8","rssi":-39,"latitude_i":538250,"longitude_i":-15663,"alias":"Living Room Ceiling Light","status":"new","description":"Smart Wi-Fi LED Bulb with Color Changing","mic_type":"IOT.SMARTBULB","mic_mac":"C0C9E379178C","dev_state":"normal","is_factory":false,"disco_ver":"1.0","ctrl_protocols":{"name":"Linkie","version":"1.0"},"active_mode":"none","is_dimmable":1,"is_color":1,"is_variable_color_temp":1,"light_state":{"on_off":0,"dft_on_state":{"mode":"normal","hue":0,"saturation":0,"color_temp":2700,"brightness":100}},"preferred_state":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":50},{"index":1,"hue":0,"saturation":100,"color_temp":0,"brightness":100},{"index":2,"hue":120,"saturation":100,"color_temp":0,"brightness":100},{"index":3,"hue":240,"saturation":100,"color_temp":0,"brightness":100}],"err_code":0}}}
	// KL130B On: {"system":{"get_sysinfo":{"sw_ver":"1.0.12 Build 210329 Rel.141126","hw_ver":"2.0","model":"KL130B(UN)","deviceId":"801211B9312B531B26C449346D30572D1DCE005F","oemId":"E45F76AD3AF13E60B58D6F68739CD7E4","hwId":"1E97141B9F0E939BD8F9679F0B6167C8","rssi":-44,"latitude_i":538250,"longitude_i":-15663,"alias":"Living Room Ceiling Light","status":"new","description":"Smart Wi-Fi LED Bulb with Color Changing","mic_type":"IOT.SMARTBULB","mic_mac":"C0C9E379178C","dev_state":"normal","is_factory":false,"disco_ver":"1.0","ctrl_protocols":{"name":"Linkie","version":"1.0"},"active_mode":"none","is_dimmable":1,"is_color":1,"is_variable_color_temp":1,"light_state":{"on_off":1,"mode":"normal","hue":0,"saturation":0,"color_temp":2700,"brightness":100},"preferred_state":[{"index":0,"hue":0,"saturation":0,"color_temp":2700,"brightness":50},{"index":1,"hue":0,"saturation":100,"color_temp":0,"brightness":100},{"index":2,"hue":120,"saturation":100,"color_temp":0,"brightness":100},{"index":3,"hue":240,"saturation":100,"color_temp":0,"brightness":100}],"err_code":0}}}
	var infoJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(deviceInfo, &infoJson); err != nil {
		return err
	}
	var data = infoJson["system"]["get_sysinfo"]
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch system info failed")
	}

	report.Common = Common{
		ActiveMode:         data["active_mode"].(string),     // e.g. "none"
		Alias:              data["alias"].(string),           // e.g. "Living Room Ceiling Light"
		ModelDescription:   mapModelDescription(model, data), // e.g. Smart Wi-Fi LED Bulb with Dimmable Light
		DeviceId:           data["deviceId"].(string),        // e.g. AABB0011CC22DD33EE44FF550011CC33FF55AA77
		HardwareId:         data["hwId"].(string),            // e.g. 00112233445566778899AA00BB00CC00
		Mac:                mapMac(model, data),              // e.g. AA0011BB2233
		ModelName:          data["model"].(string),           // e.g. KL50B(UN)
		WifiSignalStrength: int(data["rssi"].(float64)),      // e.g. -58
		DeviceType:         mapDeviceType(model, data),       // IOT.SMARTPLUGSWITCH or IOT.SMARTBULB
	}

	switch model {
	case types.KasaHS100, types.KasaHS110:
		report.SmartPlugSwitchInfo = &SmartPlugSwitchInfo{
			RelayOn:  int(data["relay_state"].(float64)) == 1,
			LedOn:    int(data["led_off"].(float64)) == 0,
			OnTime:   time.Duration(int64(data["on_time"].(float64))) * time.Second,
			Updating: int(data["updating"].(float64)) != 0,
		}
	case types.KasaKL50B, types.KasaKL110B, types.KasaKL130B:
		report.SmartLightInfo = &SmartLightInfo{
			DeviceState:                 data["dev_state"].(string), // e.g. "normal"
			IsDimmable:                  int(data["is_dimmable"].(float64)) == 1,
			IsOn:                        false,
			IsColour:                    int(data["is_color"].(float64)) == 1,
			IsVariableColourTemperature: int(data["is_variable_color_temp"].(float64)) == 1,
		}
		var lightState = data["light_state"].(map[string]interface{})
		report.SmartLightInfo.IsOn = int(lightState["on_off"].(float64)) == 1
		if report.SmartLightInfo.IsOn {
			report.SmartLightInfo.Mode = lightState["hue"].(string)
			report.SmartLightInfo.Hue = int(lightState["hue"].(float64))
			report.SmartLightInfo.Saturation = int(lightState["saturation"].(float64))
			report.SmartLightInfo.ColourTemperature = int(lightState["color_temp"].(float64))
			report.SmartLightInfo.Brightness = int(lightState["brightness"].(float64))
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
		return strings.Replace(data["mac"].(string), ":", "", -1)
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

func appendLampInfo(lampInfo []byte, report *PeriodicDeviceReport) error {
	// KL50B: {"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{"lamp_beam_angle":290,"min_voltage":220,"max_voltage":240,"wattage":7,"incandescent_equivalent":60,"max_lumens":800,"color_rendering_index":80,"err_code":0}}}
	// KL110B: {"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{"lamp_beam_angle":270,"min_voltage":110,"max_voltage":120,"wattage":10,"incandescent_equivalent":60,"max_lumens":800,"color_rendering_index":80,"err_code":0}}}
	// KL130B: {"smartlife.iot.smartbulb.lightingservice":{"get_light_details":{"lamp_beam_angle":220,"min_voltage":220,"max_voltage":240,"wattage":10,"incandescent_equivalent":60,"max_lumens":800,"color_rendering_index":80,"err_code":0}}}
	var lampJson map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(lampInfo, &lampJson); err != nil {
		return err
	}
	var data = lampJson["smartlife.iot.smartbulb.lightingservice"]["get_light_details"]
	if int(data["err_code"].(float64)) != 0 {
		return errors.New("call to fetch lamp info data failed")
	}
	if report.SmartLightInfo == nil {
		report.SmartLightInfo = &SmartLightInfo{}
	}
	report.SmartLightInfo.LampBeamAngle = int(data["lamp_beam_angle"].(float64))
	report.SmartLightInfo.MinimumVoltage = int(data["min_voltage"].(float64))
	report.SmartLightInfo.MaximumVoltage = int(data["max_voltage"].(float64))
	report.SmartLightInfo.Wattage = int(data["wattage"].(float64))
	report.SmartLightInfo.IncandescentEquivalent = int(data["incandescent_equivalent"].(float64))
	report.SmartLightInfo.MaximumLumens = int(data["max_lumens"].(float64))
	return nil
}

func appendEMeterInfo(model types.DeviceType, realTime []byte, report *PeriodicDeviceReport) error {
	// HS110: {"emeter":{"get_realtime":{"voltage_mv":246960,"current_ma":225,"power_mw":23387,"total_wh":114,"err_code":0}}}
	// KL50B: {"smartlife.iot.common.emeter":{"get_realtime":{"voltage_mv":0,"current_ma":0,"power_mw":0,"total_wh":25,"err_code":0}}}
	// KL110B: {"smartlife.iot.common.emeter":{"get_realtime":{"power_mw":3300,"err_code":0}}}
	// KL130B: {"smartlife.iot.common.emeter":{"get_realtime":{"power_mw":10800,"total_wh":44,"err_code":0}}}
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

	report.EnergyMeterInfo = &EnergyMeterInfo{
		PowerMilliWatts: int(data["power_mw"].(float64)),
	}
	if voltage, prs := data["voltage_mv"]; prs {
		report.EnergyMeterInfo.VoltageMilliVolts = int(voltage.(float64))
	}
	if current, prs := data["current_ma"]; prs {
		report.EnergyMeterInfo.CurrentMilliAmps = int(current.(float64))
	}
	if totalEnergy, prs := data["total_wh"]; prs {
		report.EnergyMeterInfo.TotalEnergyWattHours = int(totalEnergy.(float64))
	}
	return nil
}

type PeriodicDeviceReport struct {
	Common
	*SmartLightInfo
	*SmartPlugSwitchInfo
	*EnergyMeterInfo
}

type Common struct {
	ActiveMode         string
	Alias              string
	ModelDescription   string
	DeviceId           string
	HardwareId         string
	Mac                string
	ModelName          string
	WifiSignalStrength int
	DeviceType         string
	ScrapeDuration     time.Duration
}

type SmartPlugSwitchInfo struct {
	RelayOn  bool
	LedOn    bool
	OnTime   time.Duration
	Updating bool
}

type SmartLightInfo struct {
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

type EnergyMeterInfo struct {
	VoltageMilliVolts    int
	CurrentMilliAmps     int
	PowerMilliWatts      int
	TotalEnergyWattHours int
}
