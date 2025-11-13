package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var (
	_ MeteoTrentino = (*meteotrentino)(nil)

	ErrParsing = fmt.Errorf("parsing error")
)

type MeteoTrentino interface {
	FetchData(ctx context.Context) error
	Temperature() float32
	Humidity() uint8
	Rains() uint8
	Radiations() uint8
}

type meteotrentino struct {
	client                      *http.Client
	stationLastDataUrl          string
	temperature                 float32
	rains, radiations, humidity uint8
}

func NewMeteoTrentino(code string) (MeteoTrentino, error) {
	httpClient := &http.Client{}

	u, err := url.Parse("http://dati.meteotrentino.it/service.asmx/getLastDataOfMeteoStation")
	if err != nil {
		return nil, errors.Join(ErrParsing, err)
	}

	q := u.Query()
	q.Add("codice", code)
	u.RawQuery = q.Encode()

	return &meteotrentino{
		client:             httpClient,
		stationLastDataUrl: u.String(),
	}, nil
}

func (m *meteotrentino) FetchData(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.stationLastDataUrl, bytes.NewBuffer([]byte{}))
	if err != nil {
		return err
	}

	response, err := m.client.Do(req)
	if err != nil {
		return err
	}

	fmt.Println(response)

	return nil
}

func (m *meteotrentino) Temperature() float32 {
	return m.temperature
}

func (m *meteotrentino) Humidity() uint8 {
	return m.humidity
}

func (m *meteotrentino) Rains() uint8 {
	return m.rains
}

func (m *meteotrentino) Radiations() uint8 {
	return m.radiations
}
