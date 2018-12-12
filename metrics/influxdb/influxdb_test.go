package influxdb

import (
	"github.com/vitelabs/go-vite/metrics"
	"testing"
	"time"
)

func TestInfluxDBWithTags(t *testing.T) {
	metrics.InitMetrics(true, true)
	var (
		influxDbDuration  = 10 * time.Second
		influxDbEndpoint  = "http://127.0.0.1:8086"
		influxDbDatabase  = "metrics"
		influxDbUsername  = "test"
		influxDbPassword  = "test"
		influxDbNamespace = "basic/"
	)

	if metrics.InfluxDBExportFlag == true {
		if metrics.MetricsEnabled == true {
			// Start system runtime metrics collection
			go metrics.CollectProcessMetrics(3 * time.Second)
		}
		t.Log("Enabling metrics export to InfluxDB\n")
		go InfluxDBWithTags(metrics.DefaultRegistry, influxDbDuration,
			influxDbEndpoint, influxDbDatabase,
			influxDbUsername, influxDbPassword,
			influxDbNamespace, map[string]string{"host": "localhost"})

	} else {
		//log.Info("InfluxDBExport enable")
		t.Log("InfluxDBExport disable\n")
	}
}
