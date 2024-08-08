package tapo

import (
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"homepower/types"
	"testing"
)

func TestP100KlapDevice(t *testing.T) {
	server := &klapServer{
		t:        t,
		username: "test@example.com",
		password: "test_password",
		handler:  handleP100,
	}
	testServer, port := createKlapServer(t, server)
	defer testServer.Close()

	device, err := NewDevice(server.username, server.password, &types.DeviceConfig{
		Name:  "Test Device",
		Room:  "Room",
		Model: types.TapoP100,
		Ip:    "127.0.0.1",
	}, prometheus.NewRegistry(), port)
	assert.NoError(t, err)
	assert.NotNil(t, device)

	err = device.PollDeviceAndUpdateMetrics()
	assert.NoError(t, err)
}

func TestP110KlapDevice(t *testing.T) {
	server := &klapServer{
		t:        t,
		username: "test@example.com",
		password: "test_password",
		handler:  handleKlapP110Original,
	}
	testServer, port := createKlapServer(t, server)
	defer testServer.Close()

	device, err := NewDevice(server.username, server.password, &types.DeviceConfig{
		Name:  "Test Device",
		Room:  "Room",
		Model: types.TapoP110,
		Ip:    "127.0.0.1",
	}, prometheus.NewRegistry(), port)
	assert.NoError(t, err)
	assert.NotNil(t, device)

	err = device.PollDeviceAndUpdateMetrics()
	assert.NoError(t, err)
}

func TestP110KlapDeviceAugust2024(t *testing.T) {
	server := &klapServer{
		t:        t,
		username: "test@example.com",
		password: "test_password",
		handler:  handleKlapP110August2024,
	}
	testServer, port := createKlapServer(t, server)
	defer testServer.Close()

	device, err := NewDevice(server.username, server.password, &types.DeviceConfig{
		Name:  "Test Device",
		Room:  "Room",
		Model: types.TapoP110,
		Ip:    "127.0.0.1",
	}, prometheus.NewRegistry(), port)
	assert.NoError(t, err)
	assert.NotNil(t, device)

	err = device.PollDeviceAndUpdateMetrics()
	assert.NoError(t, err)
}

func handleKlapP100(t *testing.T, method string, params any) ([]byte, error) {
	t.Logf("Method: %s, Params: %v", method, params)
	if method == "get_device_info" {
		return json.Marshal(struct {
			ErrorCode int `json:"error_code"`
			Result    any `json:"result"`
		}{
			ErrorCode: 0,
			Result: struct {
				DeviceId           string  `json:"device_id"`
				FwVer              string  `json:"fw_ver"`
				HwVer              string  `json:"hw_ver"`
				Type               string  `json:"type"`
				Model              string  `json:"model"`
				MacAddress         string  `json:"mac"`
				HwId               string  `json:"hw_id"`
				FwId               string  `json:"fw_id"`
				OemId              string  `json:"oem_id"`
				Specs              string  `json:"specs"`
				DeviceOn           bool    `json:"device_on"`
				OnTime             int     `json:"on_time"`
				Overheated         bool    `json:"overheated"`
				Nickname           string  `json:"nickname"`
				Location           string  `json:"location"`
				Avatar             string  `json:"avatar"`
				Longitude          float64 `json:"longitude"`
				Latitude           float64 `json:"latitude"`
				HasSetLocationInfo bool    `json:"has_set_location_info"`
				IP                 string  `json:"ip"`
				Ssid               string  `json:"ssid"`
				SignalLevel        int     `json:"signal_level"`
				Rssi               int     `json:"rssi"`
				Region             string  `json:"region"`
				TimeDiff           int     `json:"time_diff"`
				Lang               string  `json:"lang"`
				DefaultStates      any     `json:"default_states"`
				AutoOffStatus      string  `json:"auto_off_status"`
				AutoOffRemainTime  int     `json:"auto_off_remain_time"`
			}{
				DeviceId:           "802111122223333444455556666777788889999A",
				FwVer:              "1.5.5 Build 20230927 Rel. 40646",
				HwVer:              "1.20.0",
				Type:               "SMART.TAPOPLUG",
				Model:              "P100",
				MacAddress:         "AA-BB-CC-11-22-33",
				HwId:               "999888777666555444333222111000AA",
				FwId:               "13131313A1A1A1A1F8F8F8F859595959",
				OemId:              "A3B2C1A3B2C1A3B2C1A3B2C1A3B2C1A3",
				Specs:              "UK",
				DeviceOn:           false,
				OnTime:             0,
				Overheated:         false,
				Nickname:           "U2xvdyBDb29rZXI=",
				Location:           "",
				Avatar:             "egg_boiler",
				Longitude:          -1879048193,
				Latitude:           -1879048193,
				HasSetLocationInfo: false,
				IP:                 "127.0.0.1",
				Ssid:               "QWxleElvVA==",
				SignalLevel:        3,
				Rssi:               -44,
				Region:             "Europe/London",
				TimeDiff:           0,
				Lang:               "en_US",
				DefaultStates: struct {
					State any    `json:"state"`
					Type  string `json:"type"`
				}{
					Type: "custom",
					State: map[string]bool{
						"on": false,
					},
				},
				AutoOffStatus:     "off",
				AutoOffRemainTime: 0,
			},
		})
	} else {
		return nil, errors.New("method not known: " + method)
	}
}

func handleKlapP110Original(t *testing.T, method string, params any) ([]byte, error) {
	t.Logf("Method: %s, Params: %v", method, params)
	if method == "get_device_info" {
		return json.Marshal(struct {
			ErrorCode int `json:"error_code"`
			Result    any `json:"result"`
		}{
			ErrorCode: 0,
			Result: struct {
				DeviceId           string  `json:"device_id"`
				FwVer              string  `json:"fw_ver"`
				HwVer              string  `json:"hw_ver"`
				Type               string  `json:"type"`
				Model              string  `json:"model"`
				MacAddress         string  `json:"mac"`
				HwId               string  `json:"hw_id"`
				FwId               string  `json:"fw_id"`
				OemId              string  `json:"oem_id"`
				Specs              string  `json:"specs"`
				DeviceOn           bool    `json:"device_on"`
				OnTime             int     `json:"on_time"`
				Overheated         bool    `json:"overheated"`
				Nickname           string  `json:"nickname"`
				Avatar             string  `json:"avatar"`
				Longitude          float64 `json:"longitude"`
				Latitude           float64 `json:"latitude"`
				HasSetLocationInfo bool    `json:"has_set_location_info"`
				IP                 string  `json:"ip"`
				Ssid               string  `json:"ssid"`
				SignalLevel        int     `json:"signal_level"`
				Rssi               int     `json:"rssi"`
				Region             string  `json:"region"`
				TimeDiff           int     `json:"time_diff"`
				Lang               string  `json:"lang"`
				DefaultStates      any     `json:"default_states"`
			}{
				DeviceId:           "802111122223333444455556666777788889999A",
				FwVer:              "1.0.7 Build 210629 Rel.174901",
				HwVer:              "1.0",
				Type:               "SMART.TAPOPLUG",
				Model:              "P110",
				MacAddress:         "AA-BB-CC-11-22-33",
				HwId:               "999888777666555444333222111000AA",
				FwId:               "00000000000000000000000000000000",
				OemId:              "A3B2C1A3B2C1A3B2C1A3B2C1A3B2C1A3",
				Specs:              "",
				DeviceOn:           true,
				OnTime:             2386,
				Overheated:         false,
				Nickname:           "RnJpZGdlIEZyZWV6ZXIg", // base64 for "Fridge Freezer " with the trailing space
				Avatar:             "fan",
				Longitude:          -501234, // degrees * 1000, smudged for privacy
				Latitude:           -11234,
				HasSetLocationInfo: true,
				IP:                 "127.0.0.1",
				Ssid:               "QWxleElvVA==",
				SignalLevel:        2,
				Rssi:               -56,
				Region:             "Europe/London",
				TimeDiff:           0,
				Lang:               "en_US",
				DefaultStates: struct {
					State any    `json:"state"`
					Type  string `json:"type"`
				}{
					Type: "custom",
					State: map[string]bool{
						"on": true,
					},
				},
			},
		})
	} else if method == "get_energy_usage" {
		return json.Marshal(struct {
			ErrorCode int `json:"error_code"`
			Result    any `json:"result"`
		}{
			ErrorCode: 0,
			Result: struct {
				CurrentPower int     `json:"current_power"`
				LocalTime    string  `json:"local_time"`
				MonthEnergy  int     `json:"month_energy"`
				MonthRuntime int     `json:"month_runtime"`
				Past1Year    []int   `json:"past1_year"`
				Past24Hours  []int   `json:"past24_hours"`
				Past30Days   []int   `json:"past30_days"`
				Past7Days    [][]int `json:"past7_days"`
				TodayEnergy  int     `json:"today_energy"`
				TodayRuntime int     `json:"today_runtime"`
			}{
				CurrentPower: 2529,
				LocalTime:    "2022-09-20 03:05:19",
				MonthEnergy:  5203,
				MonthRuntime: 17644,
				Past1Year:    []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5203},
				Past24Hours:  []int{14, 17, 14, 15, 14, 20, 13, 15, 14, 15, 17, 26, 16, 21, 17, 13, 14, 15, 15, 14, 23, 23, 21, 0},
				Past7Days: [][]int{
					[]int{15, 26, 17, 23, 12, 27, 17, 13, 16, 26, 14, 14, 17, 28, 15, 15, 19, 18, 29, 17, 16, 13, 20, 26},
					[]int{20, 12, 14, 14, 21, 22, 14, 14, 21, 19, 13, 20, 20, 20, 16, 14, 15, 15, 19, 21, 22, 18, 20, 13},
					[]int{23, 18, 21, 26, 14, 14, 23, 22, 13, 17, 22, 21, 22, 19, 12, 25, 13, 17, 15, 25, 23, 18, 14, 20},
					[]int{12, 14, 14, 24, 18, 17, 20, 20, 16, 13, 14, 20, 14, 26, 18, 13, 14, 14, 21, 20, 22, 21, 20, 19},
					[]int{13, 20, 20, 12, 19, 13, 19, 13, 24, 17, 13, 18, 13, 15, 25, 18, 14, 14, 17, 17, 18, 16, 13, 17},
					[]int{21, 18, 15, 17, 14, 17, 14, 15, 14, 20, 13, 15, 14, 15, 17, 26, 16, 21, 17, 13, 14, 15, 15, 14},
					[]int{23, 23, 21, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				},
				TodayEnergy:  67,
				TodayRuntime: 181,
			},
		})
	} else {
		return nil, errors.New("method not known: " + method)
	}
}

func handleKlapP110August2024(t *testing.T, method string, params any) ([]byte, error) {
	t.Logf("Method: %s, Params: %v", method, params)
	if method == "get_device_info" {
		return json.Marshal(struct {
			ErrorCode int `json:"error_code"`
			Result    any `json:"result"`
		}{
			ErrorCode: 0,
			Result: struct {
				DeviceId              string `json:"device_id"`
				FwVer                 string `json:"fw_ver"`
				HwVer                 string `json:"hw_ver"`
				Type                  string `json:"type"`
				Model                 string `json:"model"`
				MacAddress            string `json:"mac"`
				HwId                  string `json:"hw_id"`
				FwId                  string `json:"fw_id"`
				OemId                 string `json:"oem_id"`
				IP                    string `json:"ip"`
				TimeDiff              int    `json:"time_diff"`
				Ssid                  string `json:"ssid"`
				Rssi                  int    `json:"rssi"`
				SignalLevel           int    `json:"signal_level"`
				AutoOffStatus         string `json:"auto_off_status"`
				AutoOffRemainTime     int    `json:"auto_off_remain_time"`
				Lang                  string `json:"lang"`
				Avatar                string `json:"avatar"`
				Region                string `json:"region"`
				Specs                 string `json:"specs"`
				Nickname              string `json:"nickname"`
				HasSetLocationInfo    bool   `json:"has_set_location_info"`
				DeviceOn              bool   `json:"device_on"`
				OnTime                int    `json:"on_time"`
				DefaultStates         any    `json:"default_states"`
				OverheatedStatus      string `json:"overheat_status"`
				PowerProtectionStatus string `json:"power_protection_status"`
				OverCurrentStatus     string `json:"overcurrent_status"`
				ChargingStatus        string `json:"charging_status"`
			}{
				DeviceId:           "802111122223333444455556666777788889999A",
				FwVer:              "1.3.1 Build 240621 Rel.162048",
				HwVer:              "1.0",
				Type:               "SMART.TAPOPLUG",
				Model:              "P110",
				MacAddress:         "AA-BB-CC-11-22-33",
				HwId:               "999888777666555444333222111000AA",
				FwId:               "00000000000000000000000000000000",
				OemId:              "A3B2C1A3B2C1A3B2C1A3B2C1A3B2C1A3",
				IP:                 "127.0.0.1",
				TimeDiff:           0,
				Ssid:               "QWxleElvVA==",
				Rssi:               -45,
				SignalLevel:        3,
				AutoOffStatus:      "off",
				AutoOffRemainTime:  0,
				Lang:               "en_US",
				Avatar:             "fan",
				Region:             "Europe/London",
				Specs:              "",
				Nickname:           "RnJpZGdlIEZyZWV6ZXIg", // base64 for "Fridge Freezer " with the trailing space
				HasSetLocationInfo: true,
				DeviceOn:           true,
				OnTime:             2386,
				DefaultStates: struct {
					State any    `json:"state"`
					Type  string `json:"type"`
				}{
					Type: "custom",
					State: map[string]bool{
						"on": true,
					},
				},
				OverheatedStatus:      "normal",
				PowerProtectionStatus: "normal",
				OverCurrentStatus:     "normal",
				ChargingStatus:        "normal",
			},
		})
	} else if method == "get_energy_usage" {
		return json.Marshal(struct {
			ErrorCode int `json:"error_code"`
			Result    any `json:"result"`
		}{
			ErrorCode: 0,
			Result: struct {
				TodayRuntime      int    `json:"today_runtime"`
				MonthRuntime      int    `json:"month_runtime"`
				TodayEnergy       int    `json:"today_energy"`
				MonthEnergy       int    `json:"month_energy"`
				LocalTime         string `json:"local_time"`
				ElectricityCharge []int  `json:"electricity_charge"`
				CurrentPower      int    `json:"current_power"`
			}{
				TodayRuntime:      181,
				MonthRuntime:      17644,
				TodayEnergy:       67,
				MonthEnergy:       5203,
				LocalTime:         "2022-09-20 03:05:19",
				ElectricityCharge: []int{0, 0, 0},
				CurrentPower:      2529,
			},
		})
	} else {
		return nil, errors.New("method not known: " + method)
	}
}
