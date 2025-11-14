package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type WeatherStats interface {
	Temperature() float64
	Humidity() float64
	Precipitation() float64
	Radiation() float64
}

type Metrics struct {
	reg *prometheus.Registry

	temperature   prometheus.Gauge
	humidity      prometheus.Gauge
	precipitation prometheus.Gauge
	radiation     prometheus.Gauge
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		reg: reg,
		temperature: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "temperature_celsius",
			Help: "Current temperature in celsius",
		}),
		humidity: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "humidity_percent",
			Help: "Current relative humidity in percent",
		}),
		precipitation: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "precipitation_mm",
			Help: "Current precipitation in millimeters",
		}),
		radiation: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "radiation_watts_per_square_meter",
			Help: "Current radiation in watts per square meter",
		}),
	}

	reg.MustRegister(
		m.temperature,
		m.humidity,
		m.precipitation,
		m.radiation,
	)

	return m
}

func (m *Metrics) UpdateMetrics(latestMetrics WeatherStats) error {
	m.temperature.Set(latestMetrics.Temperature())
	m.humidity.Set(latestMetrics.Humidity())
	m.precipitation.Set(latestMetrics.Precipitation())
	m.radiation.Set(latestMetrics.Radiation())

	return nil
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{Registry: m.reg})
}
