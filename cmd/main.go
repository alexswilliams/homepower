package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"homepower/config"
	"homepower/device"
	"homepower/device/kasa"
	"homepower/types"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	var configs = config.StaticAppConfig

	var signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	var shouldExit = make(chan bool, 1)
	go func() { <-signals; println("Received SIGINT"); close(shouldExit) }()
	var allExited sync.WaitGroup
	allExited.Add(len(configs.Devices))

	registry := prometheus.NewRegistry()
	for _, dev := range configs.Devices {
		var lastDeviceReport = kasa.LatestDeviceReport{}
		var infoMetric = device.RegisterMetrics(registry, dev, &lastDeviceReport)
		go pollDevice(&allExited, shouldExit, dev, func(report *kasa.PeriodicDeviceReport) {
			lastDeviceReport.Latest = report
			infoMetric.With(prometheus.Labels{
				"kasa_active_mode":       report.ActiveMode,
				"kasa_alias":             report.Alias,
				"kasa_model_description": report.ModelDescription,
				"kasa_device_id":         report.DeviceId,
				"kasa_hardware_id":       report.HardwareId,
				"kasa_dev_mac":           report.Mac,
				"kasa_model_name":        report.ModelName,
				"kasa_dev_type":          report.DeviceType,
			}).Set(1.0)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go startHttpServer(mux, shouldExit)

	allExited.Wait()
	os.Exit(0)
}

func startHttpServer(mux *http.ServeMux, shouldExit chan bool) {
	server := http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       1500 * time.Millisecond,
		ReadHeaderTimeout: 500 * time.Millisecond,
		WriteTimeout:      2000 * time.Second,
	}
	println("Listening on 8080")
	go func() {
		<-shouldExit
		println("Received signal to shut down http server")
		if err := server.Shutdown(context.Background()); err != nil {
			println(err.Error())
		}
	}()
	log.Println(server.ListenAndServe())
	close(shouldExit)
}

func pollDevice(
	allExited *sync.WaitGroup,
	shouldExit <-chan bool,
	dev types.DeviceConfig,
	setLatestReport func(report *kasa.PeriodicDeviceReport),
) {
	println("Starting ticker for polling", dev.Room, dev.Name)
	defer allExited.Done()
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-shouldExit:
			println("Received should exit signal for", dev.Room, dev.Name)
			ticker.Stop()
			return
		case <-ticker.C:
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			report, err := device.ExtractAllData(&dev)
			if err == nil {
				setLatestReport(report)
			} else {
				println("Could not query", dev.Room, dev.Name, err.Error())
			}
		}
	}
}
