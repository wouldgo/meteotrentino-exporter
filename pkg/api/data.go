package api

import (
	"encoding/xml"
	"fmt"
	"time"
	"unsafe"
)

// try common layouts — the input uses "+01" (hour offset) so "-07" matches it
var layouts = []string{
	"2006-01-02T15:04:05-07", // matches 2025-11-13T00:00:00+01
	time.RFC3339,             // fallback if full RFC3339 appears
	"2006-01-02T15:04:05",    // fallback (no zone)
}

type XTime struct {
	time.Time
}

func (t *XTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var buf []byte

	if err := d.DecodeElement(&buf, &start); err != nil {
		return err
	}

	s := unsafe.String(&buf[0], len(buf))

	for _, l := range layouts {
		if tt, err := time.Parse(l, s); err == nil {
			t.Time = tt
			return nil
		}
	}

	return fmt.Errorf("parse error")
}

type temperature struct {
	UnitOfMeasure []byte  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type precipitation struct {
	UnitOfMeasure []byte  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type wind struct {
	UnitSpeed     []byte `xml:"UM_speed,attr"`
	UnitWindgust  []byte `xml:"UM_windgust,attr"`
	UnitDirection []byte `xml:"UM_direction,attr"`

	Date      XTime   `xml:"date"`
	Speed     float64 `xml:"speed_value"`
	Windgust  float64 `xml:"windgust"`
	Direction float64 `xml:"direction_value"`
}

type radiation struct {
	UnitOfMeasure []byte  `xml:"UM,attr"`
	Date          XTime   `xml:"date"`
	Value         float64 `xml:"value"`
}

type humidity struct {
	UnitOfMeasure []byte  `xml:"UM,attr"`
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

func (m *meteotrentinoResponse) Reset() {
	m.XMLName = xml.Name{}

	if m.Temperature == nil {
		m.Temperature = make([]temperature, 0, 160)
	} else {
		m.Temperature = m.Temperature[:0]
	}

	if m.Precipitation == nil {
		m.Precipitation = make([]precipitation, 0, 160)
	} else {
		m.Precipitation = m.Precipitation[:0]
	}

	if m.Wind == nil {
		m.Wind = make([]wind, 0, 160)
	} else {
		m.Wind = m.Wind[:0]
	}

	if m.Radiation == nil {
		m.Radiation = make([]radiation, 0, 160)
	} else {
		m.Radiation = m.Radiation[:0]
	}

	if m.Humidity == nil {
		m.Humidity = make([]humidity, 0, 160)
	} else {
		m.Humidity = m.Humidity[:0]
	}
}
