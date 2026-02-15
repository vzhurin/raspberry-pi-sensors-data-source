package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/host/v3"
)

const i2cBus = "1"
const bme280I2CAddress = 0x76
const metricsPort = 9101
const metricsPrefix = "sensors-1"

func main() {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	bus, err := i2creg.Open(i2cBus)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := bus.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	device, err := bmxx80.NewI2C(bus, bme280I2CAddress, &bmxx80.DefaultOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := device.Halt()
		if err != nil {
			log.Fatal(err)
		}
	}()

	collector := newPrometheusCollector(device, metricsPrefix)
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(metricsPort), nil))
}

type prometheusCollector struct {
	temperatureMetric *prometheus.Desc
	pressureMetric    *prometheus.Desc
	humidityMetric    *prometheus.Desc

	device *bmxx80.Dev
}

func newPrometheusCollector(device *bmxx80.Dev, metricsPrefix string) *prometheusCollector {
	return &prometheusCollector{
		temperatureMetric: prometheus.NewDesc(fmt.Sprintf("%s-temperature", metricsPrefix), "Shows temperature", nil, nil),
		pressureMetric:    prometheus.NewDesc(fmt.Sprintf("%s-pressure", metricsPrefix), "Shows pressure", nil, nil),
		humidityMetric:    prometheus.NewDesc(fmt.Sprintf("%s-humidity", metricsPrefix), "Shows humidity", nil, nil),

		device: device,
	}
}

func (c *prometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.temperatureMetric
	ch <- c.pressureMetric
	ch <- c.humidityMetric
}

func (c *prometheusCollector) Collect(ch chan<- prometheus.Metric) {
	var env physic.Env
	if err := c.device.Sense(&env); err != nil {
		log.Fatal(err)
	}

	temperature := prometheus.MustNewConstMetric(c.temperatureMetric, prometheus.GaugeValue, float64(env.Temperature))
	temperature = prometheus.NewMetricWithTimestamp(time.Now(), temperature)

	pressure := prometheus.MustNewConstMetric(c.pressureMetric, prometheus.GaugeValue, float64(env.Pressure))
	pressure = prometheus.NewMetricWithTimestamp(time.Now(), pressure)

	humidity := prometheus.MustNewConstMetric(c.humidityMetric, prometheus.GaugeValue, float64(env.Humidity))
	humidity = prometheus.NewMetricWithTimestamp(time.Now(), humidity)

	ch <- temperature
	ch <- pressure
	ch <- humidity
}
