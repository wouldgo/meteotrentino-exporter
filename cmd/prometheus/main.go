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

	"go.uber.org/zap"
	options "wouldgo.me/meteotrentino-exporter/cmd"
	"wouldgo.me/meteotrentino-exporter/pkg/api"
	prometheus_metrics "wouldgo.me/meteotrentino-exporter/pkg/metrics/prometheus"
)

func main() {
	opts := prometheus_metrics.NewPrometheusOptions()
	config, err := opts.Read()
	if err != nil {
		panic(fmt.Errorf("error on parsing options: %w", err).Error())
	}

	config.Log.Info("initialize station API", zap.String("station", config.Station))
	meteo, err := api.NewMeteoTrentino(api.MeteoTrentinoOptions{
		StationCode: config.Station,
		Logger:      config.Log,
	})
	if err != nil {
		config.Log.Fatal("error creating meteo trentino client", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	config.Log.Info("waiting for SIGTERM or SIGINT")
	defer stop()

	config.Log.Info("starting prometheus exporter", zap.String("station", config.Station))
	m, err := prometheus_metrics.NewPrometheusMetrics(prometheus_metrics.MetricsConfig{
		Api:             meteo,
		Logger:          config.Log,
		TimeoutDuration: 5 * time.Second,
	})
	if err != nil {
		config.Log.Fatal("error creating metrics", zap.Error(err))
	}

	router := http.NewServeMux()
	router.Handle("GET /metrics", m.Handler())
	router.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	go func() {
		addr := zap.String("addr", config.MetricsServer)
		config.Log.Info("listening on", addr)
		err := http.ListenAndServe(config.MetricsServer, router)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			config.Log.Fatal("error starting http server", addr, zap.Error(err))
		}
	}()

	options.RunProfiler(":8080", config.Log)

	<-ctx.Done()
	_, stop = context.WithTimeout(context.Background(), 5*time.Second)
	config.Log.Info("terminating")
	defer stop()

	config.Log.Info("bye")
	err = config.Log.Sync()
	if !errors.Is(err, syscall.EINVAL) {
		panic(err)
	}
}
