package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"homepower/config"
	"homepower/device"
	"homepower/device/kasa"
	"homepower/device/tapo"
	"homepower/types"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func main() {
	var configs = config.StaticAppConfig
	d, err := tapo.NewDevice(configs.TapoCredentials.EmailAddress, configs.TapoCredentials.Password, "192.168.1.56", 80)
	if err != nil {
		panic(err)
	}
	err = d.GetDeviceInfo()
	if err != nil {
		panic(err)
	}
	err = d.GetEnergyInfo()
	if err != nil {
		panic(err)
	}

	if true {
		return
	}

	var shouldExit = closeOnSigInt(make(chan bool, 1)) // is closed by SIGINT
	var allExited sync.WaitGroup
	allExited.Add(len(configs.Devices))

	registry := prometheus.NewRegistry()
	for _, dev := range configs.Devices {
		var lastDeviceReport = kasa.LatestDeviceReport{}
		var infoMetric = device.RegisterMetrics(registry, dev, &lastDeviceReport)
		var successMetric = prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "kasa_scrape_success",
			ConstLabels: kasa.GenerateCommonLabels(dev),
		})
		registry.MustRegister(successMetric)
		var lastInfoMetric *prometheus.GaugeVec = nil

		go pollDevice(&allExited, shouldExit, dev, func(report *kasa.PeriodicDeviceReport) {
			lastDeviceReport.Latest = report
			if report == nil {
				successMetric.Set(0.0)
			} else {
				successMetric.Set(1.0)
			}

			if lastInfoMetric != nil {
				lastInfoMetric.Reset()
				lastInfoMetric = nil
			}
			// TODO: this stuff really should be here - but until another manufacturer is added, "it works"
			if report != nil {
				thisInfoMetric := infoMetric.MustCurryWith(prometheus.Labels{
					"kasa_active_mode":       report.ActiveMode,
					"kasa_alias":             report.Alias,
					"kasa_model_description": report.ModelDescription,
					"kasa_device_id":         report.DeviceId,
					"kasa_hardware_id":       report.HardwareId,
					"kasa_dev_mac":           report.Mac,
					"kasa_model_name":        report.ModelName,
					"kasa_dev_type":          report.DeviceType,
				})
				if report.SmartLightInfo != nil {
					thisInfoMetric = thisInfoMetric.MustCurryWith(prometheus.Labels{
						"kasa_light_device_state":       report.SmartLightInfo.DeviceState,
						"kasa_light_is_dimmable":        strconv.FormatBool(report.SmartLightInfo.IsDimmable),
						"kasa_light_is_colour":          strconv.FormatBool(report.SmartLightInfo.IsColour),
						"kasa_light_is_variable_temp":   strconv.FormatBool(report.SmartLightInfo.IsVariableColourTemperature),
						"kasa_light_on_mode":            report.SmartLightInfo.Mode,
						"kasa_light_beam_angle":         strconv.Itoa(report.SmartLightInfo.LampBeamAngle),
						"kasa_light_min_voltage":        strconv.Itoa(report.SmartLightInfo.MinimumVoltage),
						"kasa_light_max_voltage":        strconv.Itoa(report.SmartLightInfo.MaximumVoltage),
						"kasa_light_wattage":            strconv.Itoa(report.SmartLightInfo.Wattage),
						"kasa_light_incandescent_equiv": strconv.Itoa(report.SmartLightInfo.IncandescentEquivalent),
						"kasa_light_max_lumens":         strconv.Itoa(report.SmartLightInfo.MaximumLumens),
					})
				} else {
					thisInfoMetric = thisInfoMetric.MustCurryWith(prometheus.Labels{
						"kasa_light_device_state":       "n/a",
						"kasa_light_is_dimmable":        "n/a",
						"kasa_light_is_colour":          "n/a",
						"kasa_light_is_variable_temp":   "n/a",
						"kasa_light_on_mode":            "n/a",
						"kasa_light_beam_angle":         "n/a",
						"kasa_light_min_voltage":        "n/a",
						"kasa_light_max_voltage":        "n/a",
						"kasa_light_wattage":            "n/a",
						"kasa_light_incandescent_equiv": "n/a",
						"kasa_light_max_lumens":         "n/a",
					})
				}
				thisInfoMetric.With(prometheus.Labels{}).Set(1.0)
				lastInfoMetric = thisInfoMetric
			}
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go startHttpServer(9981, mux, shouldExit)

	allExited.Wait()
	os.Exit(0)
}

func closeOnSigInt(channel chan bool) chan bool {
	var signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	go func() { <-signals; println("Received SIGINT"); close(channel) }()
	return channel
}

func startHttpServer(port int16, mux *http.ServeMux, shouldExit chan bool) {
	server := http.Server{
		Addr:              ":" + strconv.Itoa(int(port)),
		Handler:           mux,
		ReadTimeout:       1500 * time.Millisecond,
		ReadHeaderTimeout: 500 * time.Millisecond,
		WriteTimeout:      2000 * time.Second,
	}
	println("Listening on port " + strconv.Itoa(int(port)))
	go func() {
		<-shouldExit
		println("Received signal to shut down http server")
		if err := server.Shutdown(context.Background()); err != nil {
			println(err.Error())
		}
	}()
	log.Println(server.ListenAndServe())
}

func pollDevice(
	allExited *sync.WaitGroup,
	shouldExit <-chan bool,
	dev types.DeviceConfig,
	setLatestReport func(report *kasa.PeriodicDeviceReport),
) {
	println("Polling", dev.Room, dev.Name, "every 10 seconds")
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
			setLatestReport(report)
			if err != nil {
				setLatestReport(nil)
				if err.Error() != "unknown device type" {
					println("Could not query", dev.Room, dev.Name, err.Error())
				}
			}
		}
	}
}
