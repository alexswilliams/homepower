package types

const (
	KasaHS100 = iota
	KasaHS110
	KasaKL110B
	KasaKL130B
	KasaKL50B
	TapoL900
	TapoP100
)

type DeviceType int

type DeviceConfig struct {
	Name  string
	Room  string
	Model DeviceType
	Ip    string
}
