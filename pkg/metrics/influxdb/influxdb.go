package influxdb_metrics

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	influxdb "github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/go-playground/validator/v10"
	"github.com/influxdata/line-protocol/v2/lineprotocol"

	"go.uber.org/zap"
	"wouldgo.me/meteotrentino-exporter/pkg/api"
	"wouldgo.me/meteotrentino-exporter/pkg/metrics"
)

type MetricsConfig struct {
	Logger  *zap.Logger `validate:"required"`
	Station string      `validate:"required"`

	Database string `validate:"required"`
	Org      string
	Token    string `validate:"required"`
	Url      string `validate:"required"`
}

type InfluxDbMetrics struct {
	client *influxdb.Client
	logger *zap.Logger

	measure string
	station string
}

func NewInfluxDbMetrics(opts MetricsConfig) (*InfluxDbMetrics, error) {
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

	client, err := influxdb.New(influxdb.ClientConfig{
		Host:     opts.Url,
		Token:    opts.Token,
		Database: opts.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating influxdb client: %w", err)
	}

	return &InfluxDbMetrics{
		client:  client,
		logger:  opts.Logger,
		measure: "meteotrentino",
		station: strings.ToUpper(opts.Station),
	}, nil
}

func (i InfluxDbMetrics) Write(ctx context.Context, latestMetrics api.WeatherStats) error {
	temps := latestMetrics.Temperature()
	hums := latestMetrics.Humidity()
	prec := latestMetrics.Precipitation()
	rad := latestMetrics.Radiation()
	maxNum := max(max(max(max(0, len(temps)), len(hums)), len(prec)), len(rad))

	points := make(map[time.Time]*influxdb.Point, maxNum)
	for _, v := range temps {
		point := influxdb.NewPointWithMeasurement(i.measure).
			SetTag("station", i.station).
			SetTimestamp(v.Time()).
			SetField("temperature_celsius", v.Value())

		points[v.Time()] = point
	}

	for _, v := range hums {
		point, ok := points[v.Time()]
		if !ok {
			point = influxdb.NewPointWithMeasurement(i.measure).
				SetTag("station", i.station).
				SetTimestamp(v.Time())
		}

		point.
			SetField("humidity_percent", v.Value())

		points[v.Time()] = point
	}

	for _, v := range prec {
		point, ok := points[v.Time()]
		if !ok {
			point = influxdb.NewPointWithMeasurement(i.measure).
				SetTag("station", i.station).
				SetTimestamp(v.Time())
		}

		point.
			SetField("precipitation_mm", v.Value())

		points[v.Time()] = point
	}

	for _, v := range rad {
		point, ok := points[v.Time()]
		if !ok {
			point = influxdb.NewPointWithMeasurement(i.measure).
				SetTag("station", i.station).
				SetTimestamp(v.Time())
		}

		point.
			SetField("radiation_watts_per_square_meter", v.Value())

		points[v.Time()] = point
	}

	return i.client.WritePoints(ctx, slices.Collect(maps.Values(points)),
		influxdb.WithPrecision(lineprotocol.Second),
	)
}

func (i InfluxDbMetrics) Close() error {
	return i.client.Close()
}
