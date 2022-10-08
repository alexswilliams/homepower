package tapo

import (
	"log"
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

type DeviceDefaultState struct {
	state     map[string]bool
	stateType string
}

type DeviceStatus struct {
	Common
	*SmartPlugInfo
	*EnergyMeterInfo
}

type Common struct {
	DefaultStates      *DeviceDefaultState // default_states
	Alias              string              // nickname
	DeviceId           string              // device_id
	FirmwareId         string              // fw_id
	FirmwareVersion    string              // fw_ver
	HardwareId         string              // hw_id
	Mac                string              // mac
	ModelName          string              // model
	OemId              string              // oem_id
	Overheated         bool                // overheated
	WifiSignalStrength int                 // rssi
	SignalLevel        int                 // signal_level
	DeviceType         string              // type
	ScrapeDuration     time.Duration
}
type SmartPlugInfo struct {
	RelayOn bool          // device_on
	OnTime  time.Duration // on_time
}
type EnergyMeterInfo struct {
	PowerMilliWatts      int // current_power
	TotalEnergyWattHours int // month_energy or today_energy
}

func (dev *Device) GetDeviceInfo() error {
	if !dev.connection.isLoggedIn() {
		log.Println("Not logged in, will log in before device info request")
		if err := dev.connection.doLogin(); err != nil {
			return err
		}
	}

	responseResult, err := dev.connection.makeApiCall("get_device_info", nil)
	if err != nil {
		return err
	}
	log.Println(responseResult) // TODO: just for debugging
	//map[
	// avatar:fan
	// default_states:map[
	//  state:map[on:true]
	//  type:custom
	// ]
	// device_id:8022108E94DD9F0F5CD7CAA59D0F71901FE5D070
	// device_on:true
	// fw_id:00000000000000000000000000000000
	// fw_ver:1.0.7 Build 210629 Rel.174901
	// has_set_location_info:true
	// hw_id:56DD079101D61D400A11C4A3D41C51DA
	// hw_ver:1.0
	// ip:192.168.1.67
	// lang:en_US
	// latitude:501234 // (degrees * 1000 - smudged to protect my location...)
	// longitude:-11234 // (degrees * 1000 - smudged as above)
	// mac:28-87-BA-C8-DF-77
	// model:P110
	// nickname:RnJpZGdlIEZyZWV6ZXIg // base64 for "Fridge Freezer " with the trailing space
	// oem_id:AE7B616A7168B34151ABBCF86C88DF34
	// on_time:2386 // not hours, would be too long; not second, because that's only 39 minutes and what?
	// overheated:false
	// region:Europe/London
	// rssi:-56
	// signal_level:2
	// specs:
	// ssid:QWxleElvVA== // base64 for "AlexIoT"
	// time_diff:0
	// type:SMART.TAPOPLUG
	//]
	return nil
}

func (dev *Device) GetEnergyInfo() error {
	if !dev.connection.isLoggedIn() {
		log.Println("Not logged in, will log in before device info request")
		if err := dev.connection.doLogin(); err != nil {
			return err
		}
	}

	responseResult, err := dev.connection.makeApiCall("get_energy_usage", nil)
	if err != nil {
		return err
	}
	log.Println(responseResult) // TODO: just for debugging
	//map[
	// current_power:2529
	// local_time:2022-09-20 03:05:19
	// month_energy:5203
	// month_runtime:17644
	// past1y:[0 0 0 0 0 0 0 0 0 0 0 5203]
	// past24h:[14 17 14 15 14 20 13 15 14 15 17 26 16 21 17 13 14 15 15 14 23 23 21 0]
	// past30d:[0 0 0 0 0 0 0 0 0 0 0 0 0 5 0 0 0 212 473 459 484 489 475 453 417 457 424 398 390 67]
	// past7d:[
	//  [15 26 17 23 12 27 17 13 16 26 14 14 17 28 15 15 19 18 29 17 16 13 20 26]
	//  [20 12 14 14 21 22 14 14 21 19 13 20 20 20 16 14 15 15 19 21 22 18 20 13]
	//  [23 18 21 26 14 14 23 22 13 17 22 21 22 19 12 25 13 17 15 25 23 18 14 20]
	//  [12 14 14 24 18 17 20 20 16 13 14 20 14 26 18 13 14 14 21 20 22 21 20 19]
	//  [13 20 20 12 19 13 19 13 24 17 13 18 13 15 25 18 14 14 17 17 18 16 13 17]
	//  [21 18 15 17 14 17 14 15 14 20 13 15 14 15 17 26 16 21 17 13 14 15 15 14]
	//  [23 23 21 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]]
	// today_energy:67
	// today_runtime:181
	//]

	return nil
}
