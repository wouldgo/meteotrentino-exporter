package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"wouldgo.me/meteotrentino-exporter/pkg/metrics"
)

var (
	ErrMissingStation = errors.New("missing station value")

	stationEnv, stationEnvSet = os.LookupEnv("STATION")

	logEnvEnv, logEnvEnvSet     = os.LookupEnv("LOG_ENV")
	logLevelEnv, logLevelEnvSet = os.LookupEnv("LOG_LEVEL")

	metricsServerEnv, metricsServerEnvSet = os.LookupEnv("METRICS_SERVER")

	enableInfluxDbEnv, enableInfluxDbEnvSet = os.LookupEnv("ENABLE_INFLUXDB")

	station, logEnv, logLevel, metricsServer string
	enableInfluxDb                           bool
)

func ErrWrongParam(param string) error {
	return fmt.Errorf("wrong parameter value for %s", param)
}

type options struct {
	station, metricsServer string
	enableInfluxDb         bool

	log *zap.Logger

	influxdbConfig *metrics.InfluxDbConfig
}

func newOptions() (*options, error) {
	flag.StringVar(&station, "station", "", "station code, you can find them looking here: https://content.meteotrentino.it/dati-meteo/stazioni/dati-meteo.html")

	flag.StringVar(&logEnv, "log-env", "development", "logging enviroment type: production, development (default: development)")
	flag.StringVar(&logLevel, "log-level", "debug", "logging level: info, debug, error, ... (default: debug)")

	flag.StringVar(&metricsServer, "metrics-server", ":3000", "metrics server binding addresse <ip>:<port> (default: :3000)")

	flag.BoolVar(&enableInfluxDb, "enable-influxdb", false, "metrics will be publish to influxdb")
	flag.Parse()

	if stationEnvSet {
		station = stationEnv
	}

	if logEnvEnvSet {
		logEnv = logEnvEnv
	}

	if logLevelEnvSet {
		logLevel = logLevelEnv
	}

	if metricsServerEnvSet {
		metricsServer = metricsServerEnv
	}

	if enableInfluxDbEnvSet {
		boolValue, err := strconv.ParseBool(enableInfluxDbEnv)
		if err != nil {
			return nil, ErrWrongParam("enableInfluxDb")
		}
		enableInfluxDb = boolValue
	}

	if station == "" {
		return nil, ErrMissingStation
	}

	logger, err := log(logEnv, logLevel)
	if err != nil {
		return nil, fmt.Errorf("error logger creation: %w", err)
	}

	influxDbConfig, err := metrics.ParseInfluxDbConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading influxdb config: %w", err)
	}

	return &options{
		station:        station,
		metricsServer:  metricsServer,
		log:            logger,
		enableInfluxDb: enableInfluxDb,
		influxdbConfig: influxDbConfig,
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
