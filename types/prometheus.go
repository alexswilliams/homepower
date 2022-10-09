package types

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
	"time"
)

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

func NewGauge(registry prometheus.Registerer, commonLabels prometheus.Labels, ns, name string) *prometheus.Gauge {
	var gauge = prometheus.NewGauge(prometheus.GaugeOpts{Name: name, ConstLabels: commonLabels, Namespace: ns})
	registry.MustRegister(gauge)
	return &gauge
}

func SetIfPresent(gauge *prometheus.Gauge, value float64) {
	if gauge != nil {
		(*gauge).Set(value)
	}
}

func SetFromBool(gauge *prometheus.Gauge, value bool) {
	if value {
		SetIfPresent(gauge, 1.0)
	} else {
		SetIfPresent(gauge, 0.0)
	}
}

func SetFromInt(gauge *prometheus.Gauge, value int) {
	SetIfPresent(gauge, float64(value))
}

func SetFromDurationAsSeconds(gauge *prometheus.Gauge, value time.Duration) {
	SetIfPresent(gauge, value.Seconds())
}
