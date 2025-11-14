package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"wouldgo.me/meteotrentino-exporter/pkg/api"
)

func main() {
	opts, err := newOptions()
	if err != nil {
		panic(fmt.Errorf("error on parsing options: %w", err).Error())
	}

	logger := opts.log

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	logger.Info("waiting for SIGTERM or SIGINT")
	defer stop()

	logger.Info("starting prometheus exporter", zap.String("station", opts.station))
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	meteo, err := api.NewMeteoTrentino(api.MeteoTrentinoOptions{
		StationCode: opts.station,
	})
	if err != nil {
		logger.Fatal("error creating meteo trentino client", zap.Error(err))
	}

	router := http.NewServeMux()
	router.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	router.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	timer := time.NewTicker(time.Minute * 15)
	defer timer.Stop()

	go func() {
		logger.Debug("fetching for the first time data for station", zap.String("station", opts.station))
		err := meteo.FetchData(ctx)
		if err != nil {
			logger.Error("error fetching data for the first time", zap.Error(err))
		}

		m.temperature.Set(meteo.Temperature())
		m.humidity.Set(meteo.Humidity())
		m.precipitation.Set(meteo.Precipitation())
		m.radiation.Set(meteo.Radiation())

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				logger.Debug("fetching data for station", zap.String("station", opts.station))
				err := meteo.FetchData(ctx)
				if err != nil {
					logger.Error("error fetching data", zap.Error(err))
				}
				m.temperature.Set(meteo.Temperature())
				m.humidity.Set(meteo.Humidity())
				m.precipitation.Set(meteo.Precipitation())
				m.radiation.Set(meteo.Radiation())
			}
		}
	}()

	go func() {
		err := http.ListenAndServe(":8080", router)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("error starting http server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	_, stop = context.WithTimeout(context.Background(), 5*time.Second)
	logger.Info("terminating")
	defer stop()

	//TODO tearing down

	logger.Info("bye")
	err = logger.Sync()
	if !errors.Is(err, syscall.EINVAL) {
		panic(err)
	}
}

type Metrics struct {
	temperature   prometheus.Gauge
	humidity      prometheus.Gauge
	precipitation prometheus.Gauge
	radiation     prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
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
