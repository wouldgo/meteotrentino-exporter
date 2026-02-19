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

	logger.Info("initilize station API", zap.String("station", opts.station))
	meteo, err := api.NewMeteoTrentino(api.MeteoTrentinoOptions{
		StationCode: opts.station,
		Logger:      logger,
	})
	if err != nil {
		logger.Fatal("error creating meteo trentino client", zap.Error(err))
	}

	if enableInfluxDb {

		influxDb(logger, opts, meteo)
	} else {

		prometheus(logger, opts, meteo)
	}

	logger.Info("bye")
	err = logger.Sync()
	if !errors.Is(err, syscall.EINVAL) {
		panic(err)
	}
}

func influxDb(logger *zap.Logger, opts *options, meteo api.MeteoTrentino) {
	logger.Info("starting influxdb ingestion metrics", zap.String("station", opts.station))
	m, err := metrics.NewInfluxDbMetrics(metrics.InfluxDbOptions{
		Logger:  logger,
		Station: opts.station,

		Database: opts.influxdbConfig.Database,
		Org:      opts.influxdbConfig.Org,
		Token:    opts.influxdbConfig.Token,
		Url:      opts.influxdbConfig.Url,
	})
	if err != nil {
		logger.Fatal("error creating influxdb client metrics", zap.Error(err))
	}

	defer func() {
		err := m.Close()
		if err != nil {
			logger.Fatal("error closing influxdb client metrics", zap.Error(err))
		}
	}()

	ctx, stop := context.WithTimeout(context.Background(), time.Minute)
	defer stop()
	latestMetrics, err := meteo.FetchData(ctx)
	if err != nil {
		logger.Fatal("error fetching metrics", zap.Error(err))
	}

	err = m.Write(ctx, latestMetrics)
	if err != nil {
		logger.Fatal("error storing data", zap.Error(err))
	}
}

func prometheus(logger *zap.Logger, opts *options, meteo api.MeteoTrentino) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	logger.Info("waiting for SIGTERM or SIGINT")
	defer stop()

	logger.Info("starting prometheus exporter", zap.String("station", opts.station))
	m, err := metrics.NewPromMetrics(metrics.PromOptions{
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

	go func() {
		addr := zap.String("addr", opts.metricsServer)
		logger.Info("listening on", addr)
		err := http.ListenAndServe(opts.metricsServer, router)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("error starting http server", addr, zap.Error(err))
		}
	}()

	runProfiler(":8080", logger)

	<-ctx.Done()
	_, stop = context.WithTimeout(context.Background(), 5*time.Second)
	logger.Info("terminating")
	defer stop()
}
