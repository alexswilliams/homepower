package tapo

import (
	"time"
)

type Device struct {
	connection *deviceConnection
}

func NewDevice(email string, password string, ip string, port uint16) (*Device, error) {
	connection, err := newDeviceConnection(email, password, ip, port)
	if err != nil {
		return nil, err
	}
	return &Device{
		connection: connection,
	}, nil
}

type DeviceStatus struct {
	Common
	*SmartPlugInfo
	*SmartBulbInfo
	*EnergyMeterInfo
}

type Common struct {
	Alias              string // nickname
	DeviceId           string // device_id
	FirmwareId         string // fw_id
	FirmwareVersion    string // fw_ver
	HardwareId         string // hw_id
	Mac                string // mac
	ModelName          string // model
	OemId              string // oem_id
	Overheated         bool   // overheated
	WifiSignalStrength int    // rssi
	SignalLevel        int    // signal_level
	DeviceType         string // type
}
type DefaultSwitchStates struct {
	On        bool   // on
	StateType string // type, e.g. last_states or custom
}
type SmartPlugInfo struct {
	DefaultStates DefaultSwitchStates // default_states
	RelayOn       bool                // device_on
	OnTime        time.Duration       // on_time
}
type DefaultBulbStates struct {
	Brightness        int    // brightness
	ColourTemperature int    // color_temp
	Hue               int    // hue
	Saturation        int    // saturation
	StateType         string // type, e.g. last_states or custom
}
type SmartBulbInfo struct {
	DefaultStates     DefaultBulbStates // default_states
	Brightness        int               // brightness
	ColourTemperature int               // color_temp
	LightOn           bool              // device_on
	Hue               int               // hue
	Saturation        int               // saturation
}
type EnergyMeterInfo struct {
	PowerMilliWatts      int // current_power
	MonthEnergyWattHours int // month_energy
	TodayEnergyWattHours int // today_energy
}

func (dev *Device) PopulateDeviceInfo(status *DeviceStatus) error {
	responseResult, err := dev.connection.makeApiCall("get_device_info", nil)
	if err != nil {
		return err
	}
	// log.Println(responseResult)
	status.Alias = responseResult["nickname"].(string)
	status.DeviceId = responseResult["device_id"].(string)
	status.FirmwareId = responseResult["fw_id"].(string)
	status.FirmwareVersion = responseResult["fw_ver"].(string)
	status.HardwareId = responseResult["hw_id"].(string)
	status.Mac = responseResult["mac"].(string)
	status.ModelName = responseResult["model"].(string)
	status.OemId = responseResult["oem_id"].(string)
	status.Overheated = responseResult["overheated"].(bool)
	status.WifiSignalStrength = int(responseResult["rssi"].(float64))
	status.SignalLevel = int(responseResult["signal_level"].(float64))
	status.DeviceType = responseResult["type"].(string)

	defaultStates := responseResult["default_states"].(map[string]any)
	defaultStatesState := defaultStates["state"].(map[string]any)

	if status.DeviceType == "SMART.TAPOBULB" {
		status.SmartBulbInfo = &SmartBulbInfo{
			DefaultStates: DefaultBulbStates{
				Brightness:        int(defaultStatesState["brightness"].(float64)),
				ColourTemperature: int(defaultStatesState["color_temp"].(float64)),
				Hue:               int(defaultStatesState["hue"].(float64)),
				Saturation:        int(defaultStatesState["saturation"].(float64)),
				StateType:         defaultStates["type"].(string),
			},
			Brightness:        int(responseResult["brightness"].(float64)),
			ColourTemperature: int(responseResult["color_temp"].(float64)),
			LightOn:           responseResult["device_on"].(bool),
			Hue:               int(responseResult["hue"].(float64)),
			Saturation:        int(responseResult["saturation"].(float64)),
		}
	} else if status.DeviceType == "SMART.TAPOPLUG" {
		status.SmartPlugInfo = &SmartPlugInfo{
			DefaultStates: DefaultSwitchStates{
				StateType: defaultStates["type"].(string),
			},
			RelayOn: responseResult["device_on"].(bool),
			OnTime:  time.Duration(int64(responseResult["on_time"].(float64))) * time.Second,
		}
		if defaultState, hasDefaultState := defaultStatesState["on"]; hasDefaultState {
			status.SmartPlugInfo.DefaultStates.On = defaultState.(bool)
		}
	}

	return nil
}

func (dev *Device) GetEnergyInfo(status *DeviceStatus) error {
	responseResult, err := dev.connection.makeApiCall("get_energy_usage", nil)
	if err != nil {
		return err
	}
	// log.Println(responseResult)
	status.EnergyMeterInfo = &EnergyMeterInfo{
		PowerMilliWatts:      int(responseResult["current_power"].(float64)),
		MonthEnergyWattHours: int(responseResult["month_energy"].(float64)),
		TodayEnergyWattHours: int(responseResult["today_energy"].(float64)),
	}
	return nil
}
