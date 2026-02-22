package prometheus_metrics

import (
	"flag"
	"fmt"
	"os"

	"wouldgo.me/meteotrentino-exporter/pkg/options"
)

var (
	metricsServerEnv, metricsServerEnvSet = os.LookupEnv("METRICS_SERVER")
)

type PrometheusOptions struct {
	*options.Options
	metricsServer *string
}

type PrometheusConfig struct {
	*options.Config
	MetricsServer string
}

func NewPrometheusOptions() *PrometheusOptions {
	opts := options.NewOptions()
	var metricsServer string
	flag.StringVar(&metricsServer, "metrics-server", ":3000", "metrics server binding addresse <ip>:<port> (default: :3000)")

	return &PrometheusOptions{
		opts,
		&metricsServer,
	}
}

func (po *PrometheusOptions) Read() (*PrometheusConfig, error) {
	conf, err := po.Options.Read()
	if err != nil {
		return nil, fmt.Errorf("error on parsing prometheus metrics options: %w", err)
	}

	if metricsServerEnvSet {
		po.metricsServer = &metricsServerEnv
	}

	return &PrometheusConfig{
		conf,
		*po.metricsServer,
	}, nil
}
