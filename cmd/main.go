package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"homepower/device"
	"log"
	"net/http"
)
import "homepower/config"

func main() {
	var configs = config.StaticAppConfig
	device.ExtractAllData(&configs.Devices[15])

	promhttp.HandlerFor()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatalln(http.ListenAndServe(":8080", nil))
}
