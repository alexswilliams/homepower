package tapo

import (
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"homepower/types"
	"testing"
)

func TestKlapDevice(t *testing.T) {
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
				DeviceId:           "802280A16D601909124373211884D9081F4B1B9C",
				FwVer:              "1.5.5 Build 20230927 Rel. 40646",
				HwVer:              "1.20.0",
				Type:               "SMART.TAPOPLUG",
				Model:              "P100",
				MacAddress:         "5C-A6-E6-FE-BE-0B",
				HwId:               "9994A0A7D5B29645B8150C392284029D",
				FwId:               "1D18AD293A25ABDE41405B20C6F98816",
				OemId:              "D43E293FEA5A174CC7534285828B0D15",
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
