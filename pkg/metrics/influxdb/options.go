package influxdb_metrics

import (
	"flag"
	"fmt"
	"os"

	options "wouldgo.me/meteotrentino-exporter/cmd"
)

var (
	databaseEnv, databaseEnvSet = os.LookupEnv("INFLUXDB_DATABASE")
	orgEnv, orgEnvSet           = os.LookupEnv("INFLUXDB_ORG")
	tokenEnv, tokenEnvSet       = os.LookupEnv("INFLUXDB_TOKEN")
	urlEnv, urlEnvSet           = os.LookupEnv("INFLUXDB_URL")
)

type InfluxDbOptions struct {
	*options.Options
	database, org, token, url *string
}

type InfluxDbConfig struct {
	*options.Config
	Database, Org, Token, Url string
}

func NewInfluxDbOptions() *InfluxDbOptions {
	opts := options.NewOptions()

	var database, org, token, url string
	flag.StringVar(&database, "influxdb-database", "", "influxdb database")
	flag.StringVar(&org, "influxdb-org", "", "influxdb organization")
	flag.StringVar(&token, "influxdb-token", "", "influxdb token")
	flag.StringVar(&url, "influxdb-url", "", "influxdb url")

	return &InfluxDbOptions{
		opts,
		&database,
		&org,
		&token,
		&url,
	}
}

func (io *InfluxDbOptions) Read() (*InfluxDbConfig, error) {
	conf, err := io.Options.Read()
	if err != nil {
		return nil, fmt.Errorf("error on parsing influxdb metrics options: %w", err)
	}
	if databaseEnvSet {
		io.database = &databaseEnv
	}
	if orgEnvSet {
		io.org = &orgEnv
	}
	if tokenEnvSet {
		io.token = &tokenEnv
	}
	if urlEnvSet {
		io.url = &urlEnv
	}

	return &InfluxDbConfig{
		conf,
		*io.database,
		*io.org,
		*io.token,
		*io.url,
	}, nil
}
