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
	"strings"
	"time"
)

// MetricsEnabled is checked by the constructor functions for all of the
// standard metrics.  If it is true, the metric returned is chain stub.
//
// This global kill-switch helps quantify the observer effect and makes
// for less cluttered pprof profiles.
var (
	MetricsEnabled       = false
	InfluxDBExportEnable = false
	log                  = log15.New("module", "metrics")
)

type InfluxDBConfig struct {
	Endpoint string
	Database string
	Username string
	Password string
	HostTag  string
}

type Config struct {
	IsEnable         bool
	IsInfluxDBEnable bool
	InfluxDBInfo     *InfluxDBConfig
}

func InitMetrics(metricFlag, influxDBFlag bool) {
	MetricsEnabled = metricFlag
	InfluxDBExportEnable = influxDBFlag
}

var (
	systemRegistry = NewPrefixedChildRegistry(DefaultRegistry, "/system")
	cpuRegistry    = NewPrefixedChildRegistry(systemRegistry, "/cpu")
	memoryRegistry = NewPrefixedChildRegistry(systemRegistry, "/memory")
	diskRegistry   = NewPrefixedChildRegistry(systemRegistry, "/disk")
)

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
	memAllocs := GetOrRegisterMeter("/allocs", memoryRegistry)
	memFrees := GetOrRegisterMeter("/frees", memoryRegistry)
	memInuse := GetOrRegisterMeter("/inuse", memoryRegistry)
	memPauses := GetOrRegisterMeter("/pauses", memoryRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes Meter
	var diskReadBytesCounter, diskWriteBytesCounter Counter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = GetOrRegisterMeter("/readcount", diskRegistry)
		diskReadBytes = GetOrRegisterMeter("/readdata", diskRegistry)
		diskReadBytesCounter = GetOrRegisterCounter("/readbytes", diskRegistry)
		diskWrites = GetOrRegisterMeter("/writecount", diskRegistry)
		diskWriteBytes = GetOrRegisterMeter("/writedata", diskRegistry)
		diskWriteBytesCounter = GetOrRegisterCounter("/writebytes", diskRegistry)
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

func RefreshCpuStats(refresh time.Duration, prevProcessCPUTime float64, prevSystemCPUUsage gosigar.Cpu) (float64, gosigar.Cpu) {
	if !MetricsEnabled {
		return 0, gosigar.Cpu{}
	}
	frequency := float64(refresh / time.Second)
	numCPU := float64(runtime.NumCPU())
	curSystemCPUUsage := gosigar.Cpu{}
	var curProcessCPUTime float64

	if processCPUTimeGuage, ok := GetOrRegisterGaugeFloat64("/processtime", cpuRegistry).(*StandardGaugeFloat64); ok && processCPUTimeGuage != nil {
		curProcessCPUTime = getProcessCPUTime()
		deltaProcessCPUTime := curProcessCPUTime - prevProcessCPUTime
		processCPUTime := deltaProcessCPUTime / frequency / numCPU * 100

		processCPUTimeGuage.Update(processCPUTime)
	}
	if systemCPUUsageGuage, ok := GetOrRegisterGaugeFloat64("/sysusage", cpuRegistry).(*StandardGaugeFloat64); ok && systemCPUUsageGuage != nil {
		curSystemCPUUsage.Get()
		deltaSystemCPUUsage := curSystemCPUUsage.Delta(prevSystemCPUUsage)
		systemCPUValue := float64(deltaSystemCPUUsage.Sys+deltaSystemCPUUsage.User) / frequency / numCPU

		systemCPUUsageGuage.Update(systemCPUValue)
	}
	return curProcessCPUTime, curSystemCPUUsage
}

var (
	CodexecRegistry       = NewPrefixedChildRegistry(DefaultRegistry, "/codexec")
	BranchRegistry        = NewPrefixedChildRegistry(CodexecRegistry, "/branch")
	TimeConsumingRegistry = NewPrefixedChildRegistry(CodexecRegistry, "/timeconsuming")
)

func TimeConsuming(names []string, sinceTime time.Time) {
	if !MetricsEnabled {
		return
	}
	var name string
	for _, v := range names {
		name += "/" + strings.ToLower(v)
	}
	if timer, ok := GetOrRegisterResettingTimer(name, TimeConsumingRegistry).(*StandardResettingTimer); timer != nil && ok {
		timer.UpdateSince(sinceTime)
	}
}
