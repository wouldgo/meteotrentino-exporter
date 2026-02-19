//go:build profile

package main

import (
	"errors"
	"net/http"
	_ "net/http/pprof"

	"go.uber.org/zap"
)

func runProfiler(addr string, logger *zap.Logger) {
	if addr == "" {
		panic("you must provide a profiler addr")
	}
	go func() {
		logger.Info("profiling enabled", zap.String("addr", addr))
		err := http.ListenAndServe(addr, nil)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("error starting profiling server", zap.Error(err))
		}
	}()
}
