package types

import (
	"strconv"
	"strings"
)

const (
	KasaHS100 = iota
	KasaHS110
	KasaKL110B
	KasaKL130B
	KasaKL50B

	TapoL900
	TapoP100
	TapoP110
)

type DeviceType int

const (
	Unknown = iota
	Kasa
	Tapo
)

type DeviceDriver int

var kasaDeviceTypes = []DeviceType{KasaHS100, KasaHS110, KasaKL110B, KasaKL130B, KasaKL50B}
var tapoDeviceTypes = []DeviceType{TapoL900, TapoP100, TapoP110}
var deviceTypeIsLight = []DeviceType{KasaKL50B, KasaKL110B, KasaKL130B, TapoL900}

type DeviceConfig struct {
	Name  string
	Room  string
	Model DeviceType
	Ip    string
}

func DriverFor(deviceType DeviceType) DeviceDriver {
	if contains(kasaDeviceTypes, deviceType) {
		return Kasa
	}
	if contains(tapoDeviceTypes, deviceType) {
		return Tapo
	}
	return Unknown
}

func contains[E comparable](haystack []E, needle E) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func GenerateCommonLabels(dev *DeviceConfig) map[string]string {
	return map[string]string{
		"dev_room":      dev.Room,
		"dev_name":      dev.Name,
		"dev_ip":        dev.Ip,
		"dev_full_name": strings.TrimSpace(dev.Room + " " + dev.Name),
		"is_light":      strconv.FormatBool(isLight(dev.Model)),
	}
}

func isLight(model DeviceType) bool {
	return contains(deviceTypeIsLight, model)
}

type PollableDevice interface {
	PollDeviceAndUpdateMetrics() error
	ResetMetricsToRogueValues()
	CommonMetricLabels() map[string]string
}
