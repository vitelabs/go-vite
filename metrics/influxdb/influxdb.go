package influxdb

import (
	"github.com/influxdata/influxdb/client/v2"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/metrics"
	"time"
)

var log = log15.New("module", "influxdb")

const (
	// DefaultHost is the default host used to connect to an InfluxDB instance
	DefaultHost = "localhost"
	// DefaultPort is the default port used to connect to an InfluxDB instance
	DefaultPort = 8086
	// DefaultTimeout is the default connection timeout used to connect to an InfluxDB instance
	DefaultTimeout = 0
)

type reporter struct {
	reg      metrics.Registry
	interval time.Duration

	// addr should be of the form "http://host:port" or "http://[ipv6-host%zone]:port".
	addr     string
	database string
	username string
	password string

	namespace string
	tags      map[string]string

	client client.Client

	cache map[string]int64
}

func (r *reporter) makeClient() (err error) {
	r.client, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:               r.addr,
		Username:           r.username,
		Password:           r.password,
		UserAgent:          "",
		Timeout:            DefaultTimeout,
		InsecureSkipVerify: false,
		TLSConfig:          nil,
		Proxy:              nil,
	})
	return err
}

// InfluxDB starts a InfluxDB reporter which will post the from the given metrics.Registry at each d interval.
func InfluxDB(r metrics.Registry, d time.Duration, url, database, username, password, namespace string) {
	InfluxDBWithTags(r, d, url, database, username, password, namespace, nil)
}

// InfluxDBWithTags starts a InfluxDB reporter which will post the from the given metrics.Registry at each d interval with the specified tags
func InfluxDBWithTags(r metrics.Registry, d time.Duration, url, database, username, password, namespace string, tags map[string]string) {
	rep := &reporter{
		reg:       r,
		interval:  d,
		addr:      url,
		database:  database,
		username:  username,
		password:  password,
		namespace: namespace,
		tags:      tags,
		cache:     make(map[string]int64),
	}
	if err := rep.makeClient(); err != nil {
		log.Warn("Unable to make InfluxDB client", "err", err)
		return
	}

	rep.run()
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(time.Second * 5)

	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				log.Warn("Unable to send to InfluxDB", "err", err)
			}
		case <-pingTicker:
			_, _, err := r.client.Ping(r.interval)
			if err != nil {
				log.Warn("Got error while sending a ping to InfluxDB, trying to recreate client", "err", err)

				if err = r.makeClient(); err != nil {
					log.Warn("Unable to make InfluxDB client", "err", err)
				}
			}
		}
	}
}

// todo
func (r *reporter) send() error {
	return nil
}
