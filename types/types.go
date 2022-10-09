package types

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
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

type DeviceConfig struct {
	Name  string
	Room  string
	Model DeviceType
	Ip    string
}

func GenerateCommonLabels(dev *DeviceConfig) prometheus.Labels {
	return prometheus.Labels{
		"dev_room":      dev.Room,
		"dev_name":      dev.Name,
		"dev_ip":        dev.Ip,
		"dev_full_name": strings.TrimSpace(fmt.Sprintf("%s %s", dev.Room, dev.Name)),
		"is_light":      strconv.FormatBool(isLight(dev.Model)),
	}
}

func isLight(model DeviceType) bool {
	return model == KasaKL50B || model == KasaKL110B || model == KasaKL130B || model == TapoL900
}

type PollableDevice interface {
	UpdateMetrics() error
}
