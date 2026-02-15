package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/host/v3"
)

const i2cBus = "1"
const bme280I2CAddress = 0x76
const metricsPort = 9101

func main() {
	err := initHost()
	if err != nil {
		log.Fatal(err)
	}

	bus, err := newBus(i2cBus)
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	device, err := newDevice(bus, bme280I2CAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Halt()

	collector := newPrometheusCollector(device)
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(metricsPort), nil))
}

func initHost() error {
	// Load all the drivers:
	if _, err := host.Init(); err != nil {
		return err
	}

	return nil
}

func newBus(i2cBus string) (i2c.BusCloser, error) {
	// Open a handle to the first available I²C bus:
	bus, err := i2creg.Open(i2cBus)
	if err != nil {
		return nil, err
	}

	return bus, err
}

func newDevice(bus i2c.Bus, address uint16) (*bmxx80.Dev, error) {
	// Open a handle to a bme280/bmp280 connected on the²C bus using default
	// settings:
	device, err := bmxx80.NewI2C(bus, address, &bmxx80.DefaultOpts)
	if err != nil {
		return nil, err
	}

	return device, err
}

func newEnv(device *bmxx80.Dev) (*physic.Env, error) {
	// Read temperature from the sensor:
	var env physic.Env
	if err := device.Sense(&env); err != nil {
		return nil, err
	}

	return &env, nil
}

type prometheusCollector struct {
	temperatureMetric *prometheus.Desc
	pressureMetric    *prometheus.Desc
	humidityMetric    *prometheus.Desc

	device *bmxx80.Dev
}

func newPrometheusCollector(device *bmxx80.Dev) *prometheusCollector {
	return &prometheusCollector{
		temperatureMetric: prometheus.NewDesc("Temperature", "Shows temperature", nil, nil),
		pressureMetric:    prometheus.NewDesc("Pressure", "Shows pressure", nil, nil),
		humidityMetric:    prometheus.NewDesc("Humidity", "Shows humidity", nil, nil),

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
		panic(err)
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
