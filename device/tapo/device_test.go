package tapo

import (
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"homepower/types"
	"testing"
)

func TestOldDevice(t *testing.T) {
	server := &oldServer{
		t:        t,
		username: "test@example.com",
		password: "test_password",
		handler:  handleP100,
	}
	testServer, port := createOldServer(t, server)
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

func handleP100(t *testing.T, method string, params any) ([]byte, error) {
	t.Logf("Method: %s, Params: %v", method, params)
	if method == "get_device_info" {
		return json.Marshal(struct {
			Result    any `json:"result"`
			ErrorCode int `json:"error_code"`
		}{
			ErrorCode: 0,
			Result: struct {
				Avatar             string  `json:"avatar"`
				DeviceId           string  `json:"device_id"`
				DeviceOn           bool    `json:"device_on"`
				FwId               string  `json:"fw_id"`
				FwVer              string  `json:"fw_ver"`
				HasSetLocationInfo bool    `json:"has_set_location_info"`
				HwId               string  `json:"hw_id"`
				HwVer              string  `json:"hw_ver"`
				IP                 string  `json:"ip"`
				Lang               string  `json:"lang"`
				Latitude           float64 `json:"latitude"`
				Longitude          float64 `json:"longitude"`
				Location           string  `json:"location"`
				MacAddress         string  `json:"mac"`
				Model              string  `json:"model"`
				Nickname           string  `json:"nickname"`
				OemId              string  `json:"oem_id"`
				OnTime             int     `json:"on_time"`
				Overheated         bool    `json:"overheated"`
				Region             string  `json:"region"`
				Rssi               int     `json:"rssi"`
				SignalLevel        int     `json:"signal_level"`
				Specs              string  `json:"specs"`
				Ssid               string  `json:"ssid"`
				TimeDiff           int     `json:"time_diff"`
				Type               string  `json:"type"`
				TimeUsageToday     int     `json:"time_usage_today"`
				TimeUsagePast7     int     `json:"time_usage_past7"`
				TimeUsagePast30    int     `json:"time_usage_past30"`
				DefaultStates      any     `json:"default_states"`
			}{
				Avatar:             "plug",
				DeviceId:           "8022773CA54EB0774EC28EE59F6ECF951F4B0EDC",
				DeviceOn:           true,
				FwId:               "1D18AD293A25ABDE41405B20C6F98816",
				FwVer:              "1.2.10 Build 20210207 Rel. 67438",
				HasSetLocationInfo: false,
				HwId:               "9994A0A7D5B29645B8150C392284029D",
				HwVer:              "1.20.0",
				IP:                 "127.0.0.1",
				Lang:               "en_US",
				Latitude:           -1879048193, // rogue value - 0x8FFFFFFF
				Longitude:          -1879048193, // rogue value - 0x8FFFFFFF
				Location:           "living_room",
				MacAddress:         "5C-A6-E6-FE-C3-36",
				Model:              "P100",
				Nickname:           "U21hcnQgUGx1Zw==",
				OemId:              "D43E293FEA5A174CC7534285828B0D15",
				OnTime:             194,
				Overheated:         false,
				Region:             "Europe/London",
				Rssi:               -34,
				SignalLevel:        3,
				Specs:              "UK",
				Ssid:               "QWxleElvVA==",
				TimeDiff:           0,
				Type:               "SMART.TAPOPLUG",
				TimeUsageToday:     3,
				TimeUsagePast7:     3,
				TimeUsagePast30:    3,
				DefaultStates: struct {
					State any    `json:"state"`
					Type  string `json:"type"`
				}{
					State: map[string]bool{},
					Type:  "last_states",
				},
			},
		})
	} else {
		return nil, errors.New("method not known: " + method)
	}
}
