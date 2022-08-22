package config

import (
	"homepower/types"
)

type AllConfig struct {
	Devices []types.DeviceConfig
}

var StaticAppConfig = AllConfig{Devices: []types.DeviceConfig{
	// Lights
	{
		Name:  "Pendant Light",
		Ip:    "192.168.1.50",
		Model: types.KasaKL130B,
		Room:  "Living Room",
	}, {
		Name:  "Pendant Light",
		Ip:    "192.168.1.51",
		Model: types.KasaKL130B,
		Room:  "Den",
	}, {
		Name:  "Pendant Light",
		Ip:    "192.168.1.52",
		Model: types.KasaKL130B,
		Room:  "Kitchen",
	}, {
		Name:  "Pendant Light",
		Ip:    "192.168.1.53",
		Model: types.KasaKL130B,
		Room:  "Bedroom",
	}, {
		Name:  "Pendant Light",
		Ip:    "192.168.1.54",
		Model: types.KasaKL110B,
		Room:  "Hallway",
	}, {
		Name:  "Pendant Light",
		Ip:    "192.168.1.55",
		Model: types.KasaKL50B,
		Room:  "Office",
	}, {
		Name:  "Monitors Backlight Strip",
		Ip:    "192.168.1.56",
		Model: types.TapoL900,
		Room:  "Office",
	}, {
		Name:  "Christmas Lights",
		Ip:    "192.168.1.57",
		Model: types.KasaHS100,
		Room:  "Living Room",
	}, {
		Name:  "Up-Lighter",
		Ip:    "192.168.1.58",
		Model: types.KasaHS100,
		Room:  "Living Room",
	}, {
		Name:  "Lava Lamp",
		Ip:    "192.168.1.59",
		Model: types.KasaHS100,
		Room:  "Bedroom",
	},
	// Other stuff
	{
		Name:  "Work Desk Power",
		Ip:    "192.168.1.60",
		Model: types.KasaHS110,
		Room:  "Office",
	}, {
		Name:  "Radiator Power",
		Ip:    "192.168.1.61",
		Model: types.KasaHS110,
	}, {
		Name:  "Air Purifier",
		Ip:    "192.168.1.62",
		Model: types.KasaHS110,
	}, {
		Name:  "Kettle",
		Ip:    "192.168.1.63",
		Model: types.KasaHS110,
		Room:  "Kitchen",
	}, {
		Name:  "Slow Cooker",
		Ip:    "192.168.1.64",
		Model: types.TapoP100,
		Room:  "Kitchen",
	}, {
		Name:  "PC Desk Power",
		Ip:    "192.168.1.65",
		Model: types.KasaHS110,
		Room:  "Den",
	}, {
		Name:  "TV Power",
		Ip:    "192.168.1.66",
		Model: types.KasaHS110,
		Room:  "Living Room",
	}}}
