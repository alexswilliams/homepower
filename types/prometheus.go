package types

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

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
