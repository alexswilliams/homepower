package main

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"homepower/config"
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
	registry := prometheus.NewRegistry()

	var shouldExit = closeOnSigInt(make(chan bool, 1)) // is closed by SIGINT
	var allExited sync.WaitGroup
	allExited.Add(len(configs.Devices))

	for _, deviceConfig := range configs.Devices {
		pollableDevice, err := deviceFactory(deviceConfig, configs, registry)
		if err != nil {
			panic(err)
		}

		scrapeMetrics := registerScrapeMetrics(pollableDevice, registry)
		go pollDevice(&allExited, shouldExit, deviceConfig, pollableDevice, scrapeMetrics)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go startHttpServer(9981, mux, shouldExit)

	allExited.Wait()
	os.Exit(0)
}

func deviceFactory(deviceConfig types.DeviceConfig, configs config.AllConfig, registry prometheus.Registerer) (types.PollableDevice, error) {
	switch types.DriverFor(deviceConfig.Model) {
	case types.Kasa:
		return kasa.NewDevice(&deviceConfig, registry)
	case types.Tapo:
		return tapo.NewDevice(configs.TapoCredentials.EmailAddress, configs.TapoCredentials.Password, &deviceConfig, registry)
	default:
		return nil, errors.New("unknown device type")
	}
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

func pollDevice(allExited *sync.WaitGroup, shouldExit <-chan bool, deviceConfig types.DeviceConfig, dev types.PollableDevice, scrapeMetrics prometheusScrapeMetrics) {
	println("Polling", deviceConfig.Room, deviceConfig.Name, "every 10 seconds")
	defer allExited.Done()
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-shouldExit:
			println("Received should exit signal for", deviceConfig.Room, deviceConfig.Name)
			ticker.Stop()
			return
		case <-ticker.C:
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

			timeBefore := time.Now()
			err := dev.PollDeviceAndUpdateMetrics()
			scrapeMetrics.lastScrapeDuration.Set(time.Since(timeBefore).Seconds())

			if err == nil {
				scrapeMetrics.successes.Inc()
			} else {
				scrapeMetrics.failures.Inc()
				dev.ResetMetricsToRogueValues()
				if err.Error() != "unknown device type" {
					log.Println("Could not query", deviceConfig.Room, deviceConfig.Name, err.Error())
				}
			}
		}
	}
}

type prometheusScrapeMetrics struct {
	successes          prometheus.Counter
	failures           prometheus.Counter
	lastScrapeDuration prometheus.Gauge
}

func registerScrapeMetrics(dev types.PollableDevice, registry prometheus.Registerer) prometheusScrapeMetrics {
	successes := prometheus.NewCounter(prometheus.CounterOpts{Namespace: "common", Name: "scrape_successes_total", ConstLabels: dev.CommonMetricLabels()})
	registry.MustRegister(successes)
	failures := prometheus.NewCounter(prometheus.CounterOpts{Namespace: "common", Name: "scrape_failures_total", ConstLabels: dev.CommonMetricLabels()})
	registry.MustRegister(failures)
	lastScrapeDuration := prometheus.NewGauge(prometheus.GaugeOpts{Namespace: "common", Name: "last_scrape_duration_ms", ConstLabels: dev.CommonMetricLabels()})
	registry.MustRegister(lastScrapeDuration)
	return prometheusScrapeMetrics{
		successes:          successes,
		failures:           failures,
		lastScrapeDuration: lastScrapeDuration,
	}
}
