# Meteotrentino Prometheus Exporter

A service that periodically fetches the latest weather observations from a specified Meteo Trentino station and exposes them as Prometheus metrics.

* Fetches weather data from Meteo Trentino APIs
* Exposes metrics in Prometheus format via `/metrics`
* Health endpoint available at `/up`
* Configurable polling interval (default: 15 minutes)

## Prerequisites

- [Golang](https://golang.org/doc/install)
- [Make](https://www.gnu.org/software/make/)
- [Golangci-lint](https://golangci-lint.run/)

## How It Works

1. Starts an HTTP server exposing Prometheus metrics
2. Fetches weather information for the configured station at startup
3. Every 15 minutes (because meteotrentino updates data in interval of 15m), retrieves updated weather data
4. Updates Prometheus gauges accordingly
5. Gracefully handles shutdown via SIGTERM/SIGINT

## Endpoints

* **GET `/metrics`** – Prometheus metrics in plain text format
* **GET `/up`** – Simple liveness endpoint, returns HTTP 204

## Running the Exporter

You can run the exporter directly via `make` without manually invoking Go commands.

### Using Make

Build and run with:

```bash
make build
./_out/meteotrentino-exporter --station <station-code>
```

Or run directly (uses default station Rovereto `T0147`):

```bash
make run
```

### Using Go (without Make)

````bash
go run main.go --station <station-code>
go run main.go --station <station-code>
````

Or build it:

```bash
go build -o meteotrentino-exporter
./meteotrentino-exporter --station <station-code>
```

## Configuration Options

The exporter reads runtime configuration via flags:

* `--station` – Station code to fetch weather data for ([stations are here](https://content.meteotrentino.it/dati-meteo/stazioni/dati-meteo.html))
* `--metrics-server` – Address to bind the metrics HTTP server (e.g. `:9090`)

## Quick Start

### Build the binary

```bash
make build
```

This produces a fully static binary in `_out/meteotrentino-exporter`.

### Run the exporter

```bash
./_out/meteotrentino-exporter --station T0147
```

## Add to Prometheus

See configuration below.

## Prometheus Integration

Add the exporter as a scrape job in your Prometheus config:

```yaml
scrape_configs:
  - job_name: 'meteotrentino'
    static_configs:
      - targets: ['<fqdn_of_the_host>:9090']
```

## Metrics

The exporter exposes four Prometheus gauges representing the latest weather observations:

| Metric Name                        | Type  | Description                          |
| ---------------------------------- | ----- | ------------------------------------ |
| `temperature_celsius`              | Gauge | Current temperature in Celsius       |
| `humidity_percent`                 | Gauge | Current relative humidity (%)        |
| `precipitation_mm`                 | Gauge | Current precipitation in millimeters |
| `radiation_watts_per_square_meter` | Gauge | Solar radiation in W/m²              |

The exporter exposes gauges representing the most recent weather observations retrieved from the Meteo Trentino API.
