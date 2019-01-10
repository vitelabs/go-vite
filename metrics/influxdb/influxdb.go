package influxdb

import (
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/metrics"
	"time"
)

var log = log15.New("module", "influxdb")

// DefaultTimeout is the default connection timeout used to connect to an InfluxDB instance
const DefaultTimeout = 0

type Reporter struct {
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

	breaker      chan struct{}
	stopListener chan struct{}
}

func (r *Reporter) makeClient() (err error) {
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
		log.Info("influxdb export disable")
		return
	}
	rep, err := NewReporter(r, d, url, database, username, password, namespace, tags)
	if err != nil || rep == nil {
		log.Error("unable to make influxdb client", "err", err)
	}
	rep.run()
}

func NewReporter(r metrics.Registry, d time.Duration, url, database, username, password, namespace string, tags map[string]string) (*Reporter, error) {
	rep := &Reporter{
		reg:       r,
		interval:  d,
		addr:      url,
		database:  database,
		username:  username,
		password:  password,
		namespace: namespace,
		tags:      tags,
	}
	if err := rep.makeClient(); err != nil {
		return nil, errors.New("unable to make influxdb client, err " + err.Error())
	}
	rep.cache = make(map[string]int64)
	rep.breaker = make(chan struct{})
	rep.stopListener = make(chan struct{})

	return rep, nil
}

func (r *Reporter) run() {
	log.Info("export started")
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(r.interval)
LOOP:
	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				log.Info("unable to send to influxdb", "err", err)
			}
		case <-pingTicker:
			_, _, err := r.client.Ping(r.interval)
			if err != nil {
				log.Info("got error while sending a ping to influxdb, trying to recreate client", "err", err)

				if err = r.makeClient(); err != nil {
					log.Error("unable to make influxdb client", "err", err)
				}
			}
		case <-r.breaker:
			log.Info("call breaker")
			break LOOP
		}
	}

	log.Info("call stopListener")
	r.stopListener <- struct{}{}

	log.Info("export ended")
}

func (r *Reporter) send() error {
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
					"mean":     ms.Mean(),
					"max":      ms.Max(),
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
				fmt.Sprintf("%s%s.timer", namespace, name),
				r.tags,
				map[string]interface{}{
					"count":    ms.Count(),
					"mean":     ms.Mean(),
					"max":      ms.Max(),
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
						"mean":  t.Mean(),
						"max":   val[len(val)-1],
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

// InfluxDBWithTags starts a InfluxDB reporter which will post the from the given metrics.Registry at each d interval with the specified tags
func (r *Reporter) Start() {
	if !metrics.InfluxDBExportFlag {
		log.Error("influxdb export disable")
		return
	}
	log.Info("reporter start")
	r.run()
}

func (r *Reporter) Stop() {
	log.Info("reporter be called to stop")

	r.breaker <- struct{}{}
	close(r.breaker)

	<-r.stopListener
	close(r.stopListener)

	metrics.InfluxDBExportFlag = false

	log.Info("reporter stoped")
}
