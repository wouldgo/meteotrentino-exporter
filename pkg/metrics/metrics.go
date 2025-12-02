package metrics

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
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type MetricsOptions struct {
	Api    api.MeteoTrentino `validate:"required"`
	Logger *zap.Logger       `validate:"required"`

	TimeoutDuration time.Duration
}

type Metrics struct {
	reg     *prometheus.Registry
	api     api.MeteoTrentino
	logger  *zap.Logger
	timeout time.Duration

	temperature   prometheus.Gauge
	humidity      prometheus.Gauge
	precipitation prometheus.Gauge
	radiation     prometheus.Gauge
}

func NewMetrics(opts MetricsOptions) (*Metrics, error) {
	err := validate.Struct(opts)
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
	m := &Metrics{
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

func (m *Metrics) updateMetrics(latestMetrics api.WeatherStats) error {
	m.temperature.Set(latestMetrics.Temperature())
	m.humidity.Set(latestMetrics.Humidity())
	m.precipitation.Set(latestMetrics.Precipitation())
	m.radiation.Set(latestMetrics.Radiation())

	return nil
}

func (m *Metrics) Handler() http.Handler {
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
