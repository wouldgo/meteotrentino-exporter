package options

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

	stationEnv, stationEnvSet = os.LookupEnv("STATION")

	logEnvEnv, logEnvEnvSet     = os.LookupEnv("LOG_ENV")
	logLevelEnv, logLevelEnvSet = os.LookupEnv("LOG_LEVEL")
)

func ErrWrongParam(param string) error {
	return fmt.Errorf("wrong parameter value for %s", param)
}

type Options struct {
	station, logEnv, logLevel *string
}

type Config struct {
	Station string

	Log *zap.Logger
}

func NewOptions() *Options {
	var station, logEnv, logLevel string

	flag.StringVar(&station, "station", "", "station code, you can find them looking here: https://content.meteotrentino.it/dati-meteo/stazioni/dati-meteo.html")

	flag.StringVar(&logEnv, "log-env", "development", "logging enviroment type: production, development (default: development)")
	flag.StringVar(&logLevel, "log-level", "debug", "logging level: info, debug, error, ... (default: debug)")

	return &Options{
		&station,
		&logEnv,
		&logLevel,
	}
}

func (o *Options) Read() (*Config, error) {
	flag.Parse()

	if stationEnvSet {
		o.station = &stationEnv
	}

	if logEnvEnvSet {
		o.logEnv = &logEnvEnv
	}

	if logLevelEnvSet {
		o.logLevel = &logLevelEnv
	}

	if *o.station == "" {
		return nil, ErrMissingStation
	}

	logger, err := log(*o.logEnv, *o.logLevel)
	if err != nil {
		return nil, fmt.Errorf("error logger creation: %w", err)
	}

	return &Config{
		Station: *o.station,
		Log:     logger,
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
