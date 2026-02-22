//go:build influxdb

package main

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"go.uber.org/zap"
	"wouldgo.me/meteotrentino-exporter/pkg/api"
	influxdb_metrics "wouldgo.me/meteotrentino-exporter/pkg/metrics/influxdb"
)

func main() {
	opts := influxdb_metrics.NewInfluxDbOptions()
	config, err := opts.Read()
	if err != nil {
		panic(fmt.Errorf("error on parsing options: %w", err).Error())
	}

	config.Log.Info("initilize station API", zap.String("station", config.Station))
	meteo, err := api.NewMeteoTrentino(api.MeteoTrentinoOptions{
		StationCode: config.Station,
		Logger:      config.Log,
	})
	if err != nil {
		config.Log.Fatal("error creating meteo trentino client", zap.Error(err))
	}

	config.Log.Info("starting influxdb ingestion metrics", zap.String("station", config.Station))
	m, err := influxdb_metrics.NewInfluxDbMetrics(influxdb_metrics.MetricsConfig{
		Logger:  config.Log,
		Station: config.Station,

		Database: config.Database,
		Org:      config.Org,
		Token:    config.Token,
		Url:      config.Url,
	})
	if err != nil {
		config.Log.Fatal("error creating influxdb client metrics", zap.Error(err))
	}

	defer func() {
		err := m.Close()
		if err != nil {
			config.Log.Fatal("error closing influxdb client metrics", zap.Error(err))
		}
	}()

	ctx, stop := context.WithTimeout(context.Background(), time.Minute)
	defer stop()
	latestMetrics, err := meteo.FetchData(ctx)
	if err != nil {
		config.Log.Fatal("error fetching metrics", zap.Error(err))
	}

	err = m.Write(ctx, latestMetrics)
	if err != nil {
		config.Log.Fatal("error storing data", zap.Error(err))
	}

	config.Log.Info("bye")
	err = config.Log.Sync()
	if !errors.Is(err, syscall.EINVAL) {
		panic(err)
	}
}
