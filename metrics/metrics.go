// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>
package metrics

import (
	"github.com/elastic/gosigar"
	"github.com/vitelabs/go-vite/log15"
	"runtime"
	"time"
)

// MetricsEnabled is checked by the constructor functions for all of the
// standard metrics.  If it is true, the metric returned is a stub.
//
// This global kill-switch helps quantify the observer effect and makes
// for less cluttered pprof profiles.
var (
	MetricsEnabled     = false
	InfluxDBExportFlag = false
	log                = log15.New("module", "metrics")
)

func InitMetrics(metricFlag, influxDBFlag bool) {
	log.Info("Enabling metrics collection")
	MetricsEnabled = metricFlag
	log.Info("Enabling metrics collection and influxdb export. ")
	InfluxDBExportFlag = influxDBFlag
}

// CollectProcessMetrics periodically collects various metrics about the running process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metrics system is disabled
	if !MetricsEnabled {
		return
	}
	// Create the various data collectors
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < 2; i++ {
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
	// Define the various metrics to collect
	memAllocs := GetOrRegisterMeter("/system/memory/allocs", DefaultRegistry)
	memFrees := GetOrRegisterMeter("/system/memory/frees", DefaultRegistry)
	memInuse := GetOrRegisterMeter("/system/memory/inuse", DefaultRegistry)
	memPauses := GetOrRegisterMeter("/system/memory/pauses", DefaultRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes Meter
	var diskReadBytesCounter, diskWriteBytesCounter Counter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = GetOrRegisterMeter("/system/disk/readcount", DefaultRegistry)
		diskReadBytes = GetOrRegisterMeter("/system/disk/readdata", DefaultRegistry)
		diskReadBytesCounter = GetOrRegisterCounter("/system/disk/readbytes", DefaultRegistry)
		diskWrites = GetOrRegisterMeter("/system/disk/writecount", DefaultRegistry)
		diskWriteBytes = GetOrRegisterMeter("/system/disk/writedata", DefaultRegistry)
		diskWriteBytesCounter = GetOrRegisterCounter("/system/disk/writebytes", DefaultRegistry)
	} else {
		log.Debug("Failed to read disk metrics", "err", err)
	}

	var (
		prevProcessCPUTime = float64(0)
		prevSystemCPUUsage = gosigar.Cpu{}
	)
	prevSystemCPUUsage.Get()
	prevProcessCPUTime = getProcessCPUTime()

	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		location1 := i % 2
		location2 := (i - 1) % 2

		runtime.ReadMemStats(memstats[location1])
		memAllocs.Mark(int64(memstats[location1].Mallocs - memstats[location2].Mallocs))
		memFrees.Mark(int64(memstats[location1].Frees - memstats[location2].Frees))
		memInuse.Mark(int64(memstats[location1].Alloc - memstats[location2].Alloc))
		memPauses.Mark(int64(memstats[location1].PauseTotalNs - memstats[location2].PauseTotalNs))

		if ReadDiskStats(diskstats[location1]) == nil {
			diskReads.Mark(diskstats[location1].ReadCount - diskstats[location2].ReadCount)
			diskReadBytes.Mark(diskstats[location1].ReadBytes - diskstats[location2].ReadBytes)
			diskWrites.Mark(diskstats[location1].WriteCount - diskstats[location2].WriteCount)
			diskWriteBytes.Mark(diskstats[location1].WriteBytes - diskstats[location2].WriteBytes)

			diskReadBytesCounter.Inc(diskstats[location1].ReadBytes - diskstats[location2].ReadBytes)
			diskWriteBytesCounter.Inc(diskstats[location1].WriteBytes - diskstats[location2].WriteBytes)
		}
		prevProcessCPUTime, prevSystemCPUUsage = RefreshCpuStats(3*time.Second, prevProcessCPUTime, prevSystemCPUUsage)
		time.Sleep(refresh)
	}
}
