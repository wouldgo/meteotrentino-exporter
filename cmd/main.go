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
	"wouldgo.me/meteotrentino-exporter/pkg/api"
	"wouldgo.me/meteotrentino-exporter/pkg/metrics"
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
	meteo, err := api.NewMeteoTrentino(api.MeteoTrentinoOptions{
		StationCode: opts.station,
		Logger:      logger,
	})
	if err != nil {
		logger.Fatal("error creating meteo trentino client", zap.Error(err))
	}

	m, err := metrics.NewMetrics(metrics.MetricsOptions{
		Api:             meteo,
		Logger:          logger,
		TimeoutDuration: 5 * time.Second,
	})
	if err != nil {
		logger.Fatal("error creating metrics", zap.Error(err))
	}

	router := http.NewServeMux()
	router.Handle("GET /metrics", m.Handler())
	router.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	timer := time.NewTicker(time.Minute * 15)
	defer timer.Stop()

	go func() {
		logger.Info("metrics server", zap.String("addr", opts.metricsServer))
		err := http.ListenAndServe(opts.metricsServer, router)
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
