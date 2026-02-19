package api

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type WeatherStat interface {
	Time() time.Time
	Value() float64
}

type WeatherStats interface {
	Temperature() []WeatherStat
	Humidity() []WeatherStat
	Precipitation() []WeatherStat
	Radiation() []WeatherStat
}

var (
	validate = validator.New(validator.WithRequiredStructEnabled())

	_ MeteoTrentino = (*meteotrentino)(nil)
	_ WeatherStat   = (*meteoTrentinoStat)(nil)
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
	dataPool        sync.Pool
	readerPool      sync.Pool

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
		dataPool: sync.Pool{
			New: func() any {
				return new(meteotrentinoResponse)
			},
		},
		readerPool: sync.Pool{
			New: func() any {
				return bufio.NewReaderSize(nil, 256*1024)
			},
		},
	}, nil
}

type meteoTrentinoStat struct {
	time  time.Time
	value float64
}

func (m *meteoTrentinoStat) Time() time.Time {
	return m.time
}
func (m *meteoTrentinoStat) Value() float64 {
	return m.value
}

type meteoTrentinoStats struct {
	temperature, precipitation, radiation, humidity []WeatherStat
}

func fromMeteoTrentinoResponse(response *meteotrentinoResponse) (WeatherStats, error) {
	toReturn := &meteoTrentinoStats{
		temperature:   make([]WeatherStat, 0, len(response.Temperature)),
		precipitation: make([]WeatherStat, 0, len(response.Precipitation)),
		radiation:     make([]WeatherStat, 0, len(response.Radiation)),
		humidity:      make([]WeatherStat, 0, len(response.Humidity)),
	}
	for _, v := range response.Temperature {
		aStat := meteoTrentinoStat{
			time:  v.Date.Time,
			value: v.Value,
		}
		toReturn.temperature = append(toReturn.temperature, &aStat)
	}

	for _, v := range response.Precipitation {
		aStat := meteoTrentinoStat{
			time:  v.Date.Time,
			value: v.Value,
		}
		toReturn.precipitation = append(toReturn.precipitation, &aStat)
	}

	for _, v := range response.Radiation {
		aStat := meteoTrentinoStat{
			time:  v.Date.Time,
			value: v.Value,
		}
		toReturn.radiation = append(toReturn.radiation, &aStat)
	}

	for _, v := range response.Humidity {
		aStat := meteoTrentinoStat{
			time:  v.Date.Time,
			value: v.Value,
		}
		toReturn.humidity = append(toReturn.humidity, &aStat)
	}

	return toReturn, nil
}

func (mTS *meteoTrentinoStats) Temperature() []WeatherStat {
	return mTS.temperature
}

func (mTS *meteoTrentinoStats) Humidity() []WeatherStat {
	return mTS.humidity
}

func (mTS *meteoTrentinoStats) Precipitation() []WeatherStat {
	return mTS.precipitation
}

func (mTS *meteoTrentinoStats) Radiation() []WeatherStat {
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

	data, ok := m.dataPool.Get().(*meteotrentinoResponse)
	if !ok {
		return nil, fmt.Errorf("different struct type from data pool")
	}
	defer m.dataPool.Put(data)
	data.Reset()

	br, ok := m.readerPool.Get().(*bufio.Reader)
	if !ok {
		return nil, fmt.Errorf("different struct type from reader pool")
	}
	defer m.readerPool.Put(br)

	br.Reset(response.Body)

	decoder := xml.NewDecoder(br)

	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	for {
		tok, err := decoder.Token()

		if err == io.EOF {
			break
		}

		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "air_temperature":
				var v temperature
				err := decoder.DecodeElement(&v, &se)
				if err != nil {
					return nil, fmt.Errorf("error decoding air_temperature element: %w", err)
				}

				data.Temperature = append(data.Temperature, v)
			case "precipitation":
				var v precipitation
				err := decoder.DecodeElement(&v, &se)
				if err != nil {
					return nil, fmt.Errorf("error decoding precipitation element: %w", err)
				}

				data.Precipitation = append(data.Precipitation, v)
			case "wind10m":
				var v wind
				err := decoder.DecodeElement(&v, &se)
				if err != nil {
					return nil, fmt.Errorf("error decoding wind10m element: %w", err)
				}

				data.Wind = append(data.Wind, v)
			case "global_radiation":
				var v radiation
				err := decoder.DecodeElement(&v, &se)
				if err != nil {
					return nil, fmt.Errorf("error decoding global_radiation element: %w", err)
				}

				data.Radiation = append(data.Radiation, v)
			case "relative_humidity":
				var v humidity
				err := decoder.DecodeElement(&v, &se)
				if err != nil {
					return nil, fmt.Errorf("error decoding relative_humidity element: %w", err)
				}

				data.Humidity = append(data.Humidity, v)
			}
		}
	}

	stats, err := fromMeteoTrentinoResponse(data)
	if err != nil {
		return nil, fmt.Errorf("error converting api stats to weather stats")
	}

	return stats, nil
}
