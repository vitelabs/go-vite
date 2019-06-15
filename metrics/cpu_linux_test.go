package metrics

import (
	"github.com/elastic/gosigar"
	"testing"
	"time"
)

func TestRefreshCpuStats(t *testing.T) {
	InitMetrics(true, false)
	var (
		prevProcessCPUTime = float64(0)
		prevSystemCPUUsage = gosigar.Cpu{}
	)
	prevSystemCPUUsage.Get()
	prevProcessCPUTime = getProcessCPUTime()

	for i := 0; i < 30; i++ {
		if i%3 == 0 {
			prevProcessCPUTime, prevSystemCPUUsage = RefreshCpuStats(3*time.Second, prevProcessCPUTime, prevSystemCPUUsage)
			if systemCPUUsageGuage, ok := GetOrRegisterGaugeFloat64("/sysusage", cpuRegistry).(*StandardGaugeFloat64); ok && systemCPUUsageGuage != nil {
				t.Logf("prevProcessCPUTime:%v prevSystemCPUUsage:%v systemCPUUsageValue:%v\n", prevProcessCPUTime, prevSystemCPUUsage, systemCPUUsageGuage.Value())
			}
		}
	}
}
