package metrics

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

func BenchmarkDebugGCStats(b *testing.B) {
	r := NewRegistry()
	RegisterDebugGCStats(r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CaptureDebugGCStatsOnce(r)
	}
}

func TestDebugGCStatsBlocking(t *testing.T) {
	if g := runtime.GOMAXPROCS(0); g < 2 {
		t.Skipf("skipping TestDebugGCMemStatsBlocking with GOMAXPROCS=%d\n", g)
		return
	}
	ch := make(chan int)
	go testDebugGCStatsBlocking(ch)
	var gcStats debug.GCStats
	t0 := time.Now()
	debug.ReadGCStats(&gcStats)
	t1 := time.Now()
	t.Logf("\nt0:%v\nt1:%v\n", t0, t1)
	t.Log("i++ during debug.ReadGCStats:", <-ch)
	go testDebugGCStatsBlocking(ch)
	d := t1.Sub(t0)
	t.Log(d)
	time.Sleep(d)
	t.Log("i++ during time.Sleep:", <-ch)
}

func testDebugGCStatsBlocking(ch chan int) {
	i := 0
	for {
		select {
		case ch <- i:
			return
		default:
			i++
		}
	}
}

func TestCaptureDebugGCStats(t *testing.T) {
	r := NewRegistry()
	RegisterDebugGCStats(r)
	ch := make(chan int)
	for i := 0; i < 1; i++ {
		t0 := time.Now()
		CaptureDebugGCStatsOnce(r)
		go testDebugGCStatsBlocking(ch)
		t1 := time.Now()
		CaptureDebugGCStatsOnce(r)
		getGCStatsMeter(r)
		t.Log("i++ during debug.ReadGCStats:", <-ch)
		t.Logf("\nt0:%v\nt1:%v\n", t0, t1)
	}
}

func getGCStatsMeter(r Registry) {
	gcStats := GetOrRegisterMeter("debug.GCStats", r)
	fmt.Printf("Count:%v\nRate1:%v\nRate5:%v\nRate15:%v\nRateMean:%v\n",
		gcStats.Count(), gcStats.Rate1(), gcStats.Rate5(), gcStats.Rate15(), gcStats.RateMean())
}
