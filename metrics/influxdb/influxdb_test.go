package influxdb

import (
	"github.com/vitelabs/go-vite/metrics"
	"testing"
	"time"
)

var (
	influxDbDuration  = 10 * time.Second
	influxDbEndpoint  = "http://127.0.0.1:8086"
	influxDbDatabase  = "metrics"
	influxDbUsername  = "test"
	influxDbPassword  = "test"
	influxDbNamespace = "monitor"
)

func TestInfluxDBWithTags(t *testing.T) {
	metrics.InitMetrics(true, true)
	go metrics.CollectProcessMetrics(3 * time.Second)

	t.Log("enablening metrics export to influxdb\n")
	go InfluxDBWithTags(metrics.DefaultRegistry, influxDbDuration,
		influxDbEndpoint, influxDbDatabase,
		influxDbUsername, influxDbPassword,
		influxDbNamespace, map[string]string{"host": "localhost"})

}

func TestReporter_Start(t *testing.T) {
	metrics.InitMetrics(true, true)
	//go metrics.CollectProcessMetrics(3 * time.Second)

	t.Log("new reporter")
	rep, err := NewReporter(metrics.DefaultRegistry, influxDbDuration,
		influxDbEndpoint, influxDbDatabase,
		influxDbUsername, influxDbPassword,
		influxDbNamespace, map[string]string{"host": "localhost"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("start 1")
	rep.Start()
	time.AfterFunc(5*time.Second, func() {
		rep.Stop()
		t.Log("stop 1")
		time.AfterFunc(5*time.Second, func() {
			t.Log("start 2")
			rep.Start()
			time.AfterFunc(5*time.Second, func() {
				rep.Stop()
				t.Log("stop 2")
			})
		})
	})
}
