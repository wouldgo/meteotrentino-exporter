package api

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type WeatherStats interface {
	Temperature() float64
	Humidity() float64
	Precipitation() float64
	Radiation() float64
}

var (
	validate = validator.New(validator.WithRequiredStructEnabled())

	_ MeteoTrentino = (*meteotrentino)(nil)
	_ WeatherStats  = (*meteoTrentinoStats)(nil)

	ErrParsing   = errors.New("parsing error")
	ErrUnMarshal = fmt.Errorf("json unmarshal in error")
)

const stationLastData string = "http://dati.meteotrentino.it/service.asmx/getLastDataOfMeteoStation"

type MeteoTrentinoOptions struct {
	StationCode string      `validate:"required"`
	Logger      *zap.Logger `validate:"required"`

	TimeoutDuration time.Duration
}

type MeteoTrentino interface {
	FetchData(ctx context.Context) (WeatherStats, error)
}

type meteotrentino struct {
	client          *http.Client
	timeoutDuration time.Duration

	logger *zap.Logger

	stationLastDataUrl string
}

func NewMeteoTrentino(opts MeteoTrentinoOptions) (MeteoTrentino, error) {
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

	httpClient := &http.Client{}

	u, err := url.Parse(stationLastData)
	if err != nil {
		return nil, errors.Join(ErrParsing, err)
	}

	q := u.Query()
	q.Add("codice", opts.StationCode)
	u.RawQuery = q.Encode()

	timeoutDuration := 5 * time.Second
	if opts.TimeoutDuration != 0 {
		timeoutDuration = opts.TimeoutDuration
	}

	return &meteotrentino{
		client:             httpClient,
		timeoutDuration:    timeoutDuration,
		stationLastDataUrl: u.String(),
		logger:             opts.Logger,
	}, nil
}

type meteoTrentinoStats struct {
	temperature, precipitation, radiation, humidity float64
}

func (mTS *meteoTrentinoStats) Temperature() float64 {
	return mTS.temperature
}

func (mTS *meteoTrentinoStats) Humidity() float64 {
	return mTS.humidity
}

func (mTS *meteoTrentinoStats) Precipitation() float64 {
	return mTS.precipitation
}

func (mTS *meteoTrentinoStats) Radiation() float64 {
	return mTS.radiation
}

func (m *meteotrentino) FetchData(ctx context.Context) (WeatherStats, error) {
	m.logger.Info("fetching data from", zap.String("url", m.stationLastDataUrl))
	innerCtx, cancel := context.WithTimeout(ctx, m.timeoutDuration)
	defer cancel()
	req, err := http.NewRequestWithContext(innerCtx, http.MethodGet, m.stationLastDataUrl, nil)
	if err != nil {
		return nil, err
	}

	response, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response code: %d", response.StatusCode)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			m.logger.Warn("error closing body", zap.Error(err))
		}
	}()

	var data meteotrentinoResponse

	if err := xml.NewDecoder(response.Body).Decode(&data); err != nil {
		return nil, errors.Join(ErrUnMarshal, fmt.Errorf("failed to %s URL response body: %w", m.stationLastDataUrl, err))
	}

	return &meteoTrentinoStats{
		temperature:   data.Temperature[len(data.Temperature)-1].Value,
		precipitation: data.Precipitation[len(data.Precipitation)-1].Value,
		radiation:     data.Radiation[len(data.Radiation)-1].Value,
		humidity:      data.Humidity[len(data.Humidity)-1].Value,
	}, nil
}
