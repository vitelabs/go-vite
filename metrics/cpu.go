package metrics

import (
	"github.com/elastic/gosigar"
	"runtime"
	"syscall"
	"time"
)

var (
	cpuRegistry = NewPrefixedChildRegistry(DefaultRegistry, "/system/cpu")
)

func RefreshCpuStats(refresh time.Duration, prevProcessCPUTime float64, prevSystemCPUUsage gosigar.Cpu) (float64, gosigar.Cpu) {
	frequency := float64(refresh / time.Second)
	numCPU := float64(runtime.NumCPU())
	systemCPUUsage := gosigar.Cpu{}
	var processCPUTime float64

	if processCPUTimeGuage, ok := GetOrRegisterGaugeFloat64("/processtime", cpuRegistry).(*StandardGaugeFloat64); ok && processCPUTimeGuage != nil {
		if prevProcessCPUTime != processCPUTimeGuage.Value() {
			prevProcessCPUTime = processCPUTimeGuage.Value()
		}
		curProcessCPUTime := getProcessCPUTime()
		deltaProcessCPUTime := curProcessCPUTime - prevProcessCPUTime
		processCPUTime := deltaProcessCPUTime / frequency / numCPU * 100

		processCPUTimeGuage.Update(processCPUTime)
	}
	if systemCPUUsageGuage, ok := GetOrRegisterGaugeFloat64("/sysusage", cpuRegistry).(*StandardGaugeFloat64); ok && systemCPUUsageGuage != nil {
		systemCPUUsage.Get()
		curSystemCPUUsage := systemCPUUsage
		deltaSystemCPUUsage := curSystemCPUUsage.Delta(prevSystemCPUUsage)
		systemCPUValue := float64(deltaSystemCPUUsage.Sys+deltaSystemCPUUsage.User) / frequency / numCPU

		systemCPUUsageGuage.Update(systemCPUValue)
	}
	return processCPUTime, systemCPUUsage
}

// getProcessCPUTime retrieves the process' CPU time since program startup.
func getProcessCPUTime() float64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		log.Warn("Failed to retrieve CPU time", "err", err)
		return 0
	}
	return float64(usage.Utime.Sec+usage.Stime.Sec) + float64(usage.Utime.Usec+usage.Stime.Usec)/1000000
}
