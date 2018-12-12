package influxdb

import (
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/metrics"
	"time"
)

var log = log15.New("module", "influxdb")

// DefaultTimeout is the default connection timeout used to connect to an InfluxDB instance
const DefaultTimeout = 0

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
	if !metrics.InfluxDBExportFlag {
		fmt.Println("influxdb export disable")
		//log.Info("influxdb export disable")
		return
	}
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

func (r *reporter) send() error {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: r.database,
	})
	if err != nil {
		log.Info("unexpected error.  expected %v, actual %v", nil, err)
	}

	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()
		namespace := r.namespace
		var err error
		var pt *client.Point
		switch metric := i.(type) {
		case metrics.Counter:
			v := metric.Count()
			l := r.cache[name]
			pt, err = client.NewPoint(
				fmt.Sprintf("%s%s.count", namespace, name),
				r.tags,
				map[string]interface{}{
					"value": v - l,
				},
				now,
			)
			r.cache[name] = v
		case metrics.Gauge:
			ms := metric.Snapshot()
			pt, err = client.NewPoint(
				fmt.Sprintf("%s%s.gauge", namespace, name),
				r.tags,
				map[string]interface{}{
					"value": ms.Value(),
				},
				now,
			)
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			pt, err = client.NewPoint(
				fmt.Sprintf("%s%s.gauge", namespace, name),
				r.tags,
				map[string]interface{}{
					"value": ms.Value(),
				},
				now)
		case metrics.Histogram:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pt, err = client.NewPoint(
				fmt.Sprintf("%s%s.histogram", namespace, name),
				r.tags,
				map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
				},
				now)
		case metrics.Meter:
			ms := metric.Snapshot()
			pt, err = client.NewPoint(
				fmt.Sprintf("%s%s.meter", namespace, name),
				r.tags,
				map[string]interface{}{
					"count": ms.Count(),
					"m1":    ms.Rate1(),
					"m5":    ms.Rate5(),
					"m15":   ms.Rate15(),
					"mean":  ms.RateMean(),
				},
				now)
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pt, err = client.NewPoint(
				fmt.Sprint("%s%s.timer", namespace, name),
				r.tags,
				map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
					"m1":       ms.Rate1(),
					"m5":       ms.Rate5(),
					"m15":      ms.Rate15(),
					"meanrate": ms.RateMean(),
				},
				now)
		case metrics.ResettingTimer:
			t := metric.Snapshot()
			if len(t.Values()) > 0 {
				ps := t.Percentiles([]float64{50, 95, 99})
				val := t.Values()
				pt, err = client.NewPoint(fmt.Sprintf("%s%s.span", namespace, name),
					r.tags,
					map[string]interface{}{
						"count": len(val),
						"max":   val[len(val)-1],
						"mean":  t.Mean(),
						"min":   val[0],
						"p50":   ps[0],
						"p95":   ps[1],
						"p99":   ps[2],
					},
					now)
			}
		default:
			log.Info("Unknown metric type")
			return
		}
		if err != nil {
			log.Info("unexpected error. expected %v, actual %v", nil, err)
			return
		}
		bp.AddPoint(pt)
	})
	return r.client.Write(bp)
}


