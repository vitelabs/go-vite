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
			t.Logf("prevProcessCPUTime:%v prevSystemCPUUsage:%v\n", prevProcessCPUTime, prevSystemCPUUsage)
			prevProcessCPUTime, prevSystemCPUUsage = RefreshCpuStats(3*time.Second, prevProcessCPUTime, prevSystemCPUUsage)
		}
	}
}
