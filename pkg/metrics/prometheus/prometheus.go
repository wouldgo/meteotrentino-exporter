package prometheus_metrics

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"wouldgo.me/meteotrentino-exporter/pkg/api"
	"wouldgo.me/meteotrentino-exporter/pkg/metrics"
)

type MetricsConfig struct {
	Api    api.MeteoTrentino `validate:"required"`
	Logger *zap.Logger       `validate:"required"`

	TimeoutDuration time.Duration
}

type PrometheusMetrics struct {
	reg     *prometheus.Registry
	api     api.MeteoTrentino
	logger  *zap.Logger
	timeout time.Duration

	temperature   prometheus.Gauge
	humidity      prometheus.Gauge
	precipitation prometheus.Gauge
	radiation     prometheus.Gauge
}

func NewPrometheusMetrics(opts MetricsConfig) (*PrometheusMetrics, error) {
	err := metrics.Validate.Struct(opts)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return nil, err
		}

		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			errs := make([]error, len(validateErrs))
			for _, e := range validateErrs {
				errs = append(errs, e)
			}
			return nil, errors.Join(errs...)
		}

		return nil, err
	}

	reg := prometheus.NewRegistry()
	m := &PrometheusMetrics{
		reg:     reg,
		api:     opts.Api,
		logger:  opts.Logger,
		timeout: opts.TimeoutDuration,
		temperature: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "temperature_celsius",
			Help: "Current temperature in celsius",
		}),
		humidity: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "humidity_percent",
			Help: "Current relative humidity in percent",
		}),
		precipitation: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "precipitation_mm",
			Help: "Current precipitation in millimeters",
		}),
		radiation: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "radiation_watts_per_square_meter",
			Help: "Current radiation in watts per square meter",
		}),
	}

	reg.MustRegister(
		m.temperature,
		m.humidity,
		m.precipitation,
		m.radiation,
	)

	return m, nil
}

func (m *PrometheusMetrics) updateMetrics(latestMetrics api.WeatherStats) error {
	temp := latestMetrics.Temperature()
	hum := latestMetrics.Humidity()
	prec := latestMetrics.Precipitation()
	rad := latestMetrics.Radiation()

	m.temperature.Set(temp[len(temp)-1].Value())
	m.humidity.Set(hum[len(hum)-1].Value())
	m.precipitation.Set(prec[len(prec)-1].Value())
	m.radiation.Set(rad[len(rad)-1].Value())

	return nil
}

func (m *PrometheusMetrics) Handler() http.Handler {
	promHandler := promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{
		Registry: m.reg,
	})

	h := http.HandlerFunc(func(rsp http.ResponseWriter, req *http.Request) {
		latestStats, err := m.api.FetchData(req.Context())
		if err != nil {
			m.logger.Error("error fetching data", zap.Error(err))
		}

		err = m.updateMetrics(latestStats)
		if err != nil {
			m.logger.Error("error updating metrics", zap.Error(err))
		}

		promHandler.ServeHTTP(rsp, req)
	})

	return http.TimeoutHandler(h, m.timeout, fmt.Sprintf(
		"Exceeded configured timeout of %v.\n",
		m.timeout,
	))
}
