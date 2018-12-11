package influxdb

import (
	"fmt"
	"github.com/vitelabs/go-vite/metrics"
	"time"
)

// MetricsEnabledFlag is the CLI flag name to use to enable metrics collections.
const MetricsEnabledFlag = "metrics"
const DashboardEnabledFlag = "dashboard"
const InfluxDbExportEnabledFlag = "influxdb"

func InfluxdbExportSetup() {
	var (
		influxDbDuration  = 10 * time.Second
		influxDbEndpoint  = "http://127.0.0.1:8086"
		influxDbDatabase  = "metrics"
		influxDbUsername  = "admin"
		influxDbPassword  = "admin"
		influxDbNamespace = "basic/"
	)

	if influxDBExportFlag == true {
		//if metrics.MetricsEnabled == true {
		//	// Start system runtime metrics collection
		//	go metrics.CollectProcessMetrics(2 * time.Second)
		//}
		fmt.Print("Enabling swarm metrics export to InfluxDB\n")
		go InfluxDBWithTags(metrics.DefaultRegistry, influxDbDuration,
			influxDbEndpoint, influxDbDatabase,
			influxDbUsername, influxDbPassword,
			influxDbNamespace, map[string]string{"host": "localhost"})

	} else {
		//log.Info("InfluxDBExport enable")
		fmt.Println("InfluxDBExport enable")
	}
}

func init() {
	//var isInfluxDBEnable bool
	//flag.BoolVar(&isInfluxDBEnable, "influxdb", true, "influxdb enable")
	//flag.Parse()
	//
	//if isInfluxDBEnable == true {
	//	log.Info("Enabling metrics collection and influxdb export. ")
	//	influxDBExportFlag = true
	//	log.Info("Enabling metrics collection")
	//	metrics.MetricsEnabled = true
	//}
	log.Info("Enabling metrics collection and influxdb export. ")
	influxDBExportFlag = true
	log.Info("Enabling metrics collection")
	metrics.MetricsEnabled = true
}
