package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	logger.Info("starting prometheus exporter", zap.String("station", opts.station), zap.String("name", opts.name))

	//TODO prom logic
	meteo, err := api.NewMeteoTrentino("T0147")
	if err != nil {
		panic(err)
	}

	meteo.FetchData(ctx)

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
