package types

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

var deviceModelStringToDeviceType = map[string]DeviceType{
	"HS100":  KasaHS100,
	"HS110":  KasaHS110,
	"KL110B": KasaKL110B,
	"KL130B": KasaKL130B,
	"KL50B":  KasaKL50B,
	"L900":   TapoL900,
	"P100":   TapoP100,
	"P110":   TapoP110,
}

func DeviceTypeFor(modelName string) DeviceType {
	if deviceType, found := deviceModelStringToDeviceType[modelName]; found {
		return deviceType
	}
	panic("model name " + modelName + " does not correspond to a known device type")
}

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

type PollableDevice interface {
	PollDeviceAndUpdateMetrics() error
	ResetMetricsToRogueValues()
	ResetDeviceConnection()
	CommonMetricLabels() map[string]string
}
