package api

import (
	"encoding/xml"
	"fmt"
	"time"
)

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
