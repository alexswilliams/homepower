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
				DefaultStates      any     `json:"default_states"`
			}{
				Avatar:             "egg_boiler",
				DeviceId:           "802280A16D601909124373211884D9081F4B1B9C",
				DeviceOn:           false,
				FwId:               "1D18AD293A25ABDE41405B20C6F98816",
				FwVer:              "1.4.10 Build 20211104 Rel. 35882",
				HasSetLocationInfo: false,
				HwId:               "9994A0A7D5B29645B8150C392284029D",
				HwVer:              "1.20.0",
				IP:                 "127.0.0.1",
				Lang:               "",
				Latitude:           -1.879048193e+09, // looks like a rogue value?
				Longitude:          -1.879048193e+09,
				Location:           "",
				MacAddress:         "5C-A6-E6-FE-BE-0B",
				Model:              "P100",
				Nickname:           "U2xvdyBDb29rZXI=",
				OemId:              "D43E293FEA5A174CC7534285828B0D15",
				OnTime:             0,
				Overheated:         false,
				Region:             "Europe/London",
				Rssi:               -48,
				SignalLevel:        3,
				Specs:              "UK",
				Ssid:               "QWxleElvVA==",
				TimeDiff:           0,
				Type:               "SMART.TAPOPLUG",
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
