package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ErrMissingStation = errors.New("missing station value")
	ErrMissingName    = errors.New("missing name value")

	stationEnv, stationEnvSet = os.LookupEnv("STATION")
	nameEnv, nameEnvSet       = os.LookupEnv("NAME")

	logEnvEnv, logEnvEnvSet     = os.LookupEnv("LOG_ENV")
	logLevelEnv, logLevelEnvSet = os.LookupEnv("LOG_LEVEL")

	logEnv, logLevel, station, name string
)

type options struct {
	station, name string
	log           *zap.Logger
}

func newOptions() (*options, error) {
	flag.StringVar(&logEnv, "log-env", "development", "logging enviroment type: production, development (default: development)")
	flag.StringVar(&logLevel, "log-level", "debug", "logging level: info, debug, error, ... (default: debug)")

	flag.StringVar(&station, "station", "", "station code, you can find them looking here: https://content.meteotrentino.it/dati-meteo/stazioni/dati-meteo.html")
	flag.StringVar(&name, "name", "", "station name")

	flag.Parse()

	if stationEnvSet {
		station = stationEnv
	}

	if nameEnvSet {
		name = nameEnv
	}

	if logEnvEnvSet {
		logEnv = logEnvEnv
	}

	if logLevelEnvSet {
		logLevel = logLevelEnv
	}

	if station == "" {
		return nil, ErrMissingStation
	}

	if name == "" {
		return nil, ErrMissingName
	}

	logger, err := log(logEnv, logLevel)
	if err != nil {
		return nil, fmt.Errorf("error logger creation: %w", err)
	}

	return &options{
		station: station,
		name:    name,
		log:     logger,
	}, nil
}

func log(env, level string) (*zap.Logger, error) {
	var encoder zapcore.Encoder

	if strings.EqualFold(env, "production") {
		config := zap.NewProductionEncoderConfig()
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		config := zap.NewDevelopmentEncoderConfig()
		encoder = zapcore.NewConsoleEncoder(config)
	}

	//writer := bufio.NewWriter(os.Stderr)
	ws := zapcore.AddSync(os.Stderr)

	logLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("error level string not valid: %w", err)
	}

	core := zapcore.NewCore(encoder, ws, logLevel)
	return zap.New(core), nil
}
