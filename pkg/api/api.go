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
)

var (
	validate = validator.New(validator.WithRequiredStructEnabled())

	_ MeteoTrentino = (*meteotrentino)(nil)

	ErrParsing   = errors.New("parsing error")
	ErrUnMarshal = fmt.Errorf("json unmarshal in error")
)

const stationLastData string = "http://dati.meteotrentino.it/service.asmx/getLastDataOfMeteoStation"

type MeteoTrentinoOptions struct {
	StationCode     string `validate:"required"`
	TimeoutDuration time.Duration
}

type MeteoTrentino interface {
	FetchData(ctx context.Context) error
	Temperature() float64
	Humidity() float64
	Precipitation() float64
	Radiation() float64
}

type meteotrentino struct {
	client          *http.Client
	timeoutDuration time.Duration

	stationLastDataUrl                              string
	temperature, precipitation, radiation, humidity float64
}

type XTime struct {
	time.Time
}

func (t *XTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}

	// try common layouts — the input uses "+01" (hour offset) so "-07" matches it
	layouts := []string{
		"2006-01-02T15:04:05-07", // matches 2025-11-13T00:00:00+01
		time.RFC3339,             // fallback if full RFC3339 appears
		"2006-01-02T15:04:05",    // fallback (no zone)
	}

	var parseErr error
	for _, l := range layouts {
		if tt, err := time.Parse(l, s); err == nil {
			t.Time = tt
			return nil
		} else {
			parseErr = err
		}
	}
	return fmt.Errorf("could not parse time %q: %w", s, parseErr)
}

type temperature struct {
	UnitOfMeasure string  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type precipitation struct {
	UnitOfMeasure string  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type wind struct {
	UnitSpeed     string `xml:"UM_speed,attr"`
	UnitWindgust  string `xml:"UM_windgust,attr"`
	UnitDirection string `xml:"UM_direction,attr"`

	Date      XTime   `xml:"date"`
	Speed     float64 `xml:"speed_value"`
	Windgust  float64 `xml:"windgust"`
	Direction float64 `xml:"direction_value"`
}

type radiation struct {
	UnitOfMeasure string  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type humidity struct {
	UnitOfMeasure string  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type meteotrentinoResponse struct {
	XMLName       xml.Name        `xml:"lastData"`
	Temperature   []temperature   `xml:"temperature_list>air_temperature"`
	Precipitation []precipitation `xml:"precipitation_list>precipitation"`
	Wind          []wind          `xml:"wind_list>wind10m"`
	Radiation     []radiation     `xml:"global_radiation_list>global_radiation"`
	Humidity      []humidity      `xml:"relative_humidity_list>relative_humidity"`
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
	}, nil
}

func (m *meteotrentino) FetchData(ctx context.Context) error {
	innerCtx, cancel := context.WithTimeout(ctx, m.timeoutDuration)
	defer cancel()
	req, err := http.NewRequestWithContext(innerCtx, http.MethodGet, m.stationLastDataUrl, nil)
	if err != nil {
		return err
	}

	response, err := m.client.Do(req)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d", response.StatusCode)
	}
	defer response.Body.Close()

	var data meteotrentinoResponse

	if err := xml.NewDecoder(response.Body).Decode(&data); err != nil {
		return errors.Join(ErrUnMarshal, fmt.Errorf("failed to %s URL response body: %w", m.stationLastDataUrl, err))
	}

	m.temperature = data.Temperature[len(data.Temperature)-1].Value
	m.precipitation = data.Precipitation[len(data.Precipitation)-1].Value
	m.radiation = data.Radiation[len(data.Radiation)-1].Value
	m.humidity = data.Humidity[len(data.Humidity)-1].Value

	return nil
}

func (m *meteotrentino) Temperature() float64 {
	return m.temperature
}

func (m *meteotrentino) Humidity() float64 {
	return m.humidity
}

func (m *meteotrentino) Precipitation() float64 {
	return m.precipitation
}

func (m *meteotrentino) Radiation() float64 {
	return m.radiation
}
