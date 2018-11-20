package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"time"
)

type tpsOption struct {
	name          string
	printPerCount uint64
}

type tps struct {
	option tpsOption

	count     uint64
	startTime *time.Time
	stopTime  *time.Time

	lastPrintTime  time.Time
	lastPrintRound uint64
	lastPrintOps   uint64
}

func newTps(option tpsOption) *tps {
	op := option
	if op.name == "" {
		op.name = "tps statistics"
	}

	return &tps{
		option: option,
	}
}

func (t *tps) Start() {
	now := time.Now()
	t.startTime = &now
	t.lastPrintTime = now
}

func (t *tps) do(count uint64) {
	if t.startTime == nil {
		log15.Crit("Must call `Start` function before call `doOne` function", "method", "tps/doOne")
	}
	t.count += count
	currentPrintRound := t.count / t.option.printPerCount
	if t.option.printPerCount > 0 &&
		currentPrintRound > t.lastPrintRound {
		now := time.Now()
		fmt.Printf("%s %d ops. has done %d ops \n", t.print(t.lastPrintTime, now, t.option.printPerCount), t.Ops()-t.lastPrintOps, t.Ops())
		t.lastPrintTime = now
		t.lastPrintRound = currentPrintRound
		t.lastPrintOps = t.Ops()
	}
}

func (t *tps) doOne() {
	t.do(1)
}

func (t *tps) Stop() {
	now := time.Now()
	t.stopTime = &now
}

func (t *tps) Ops() uint64 {
	return t.count
}

func (t *tps) print(startTime time.Time, stopTime time.Time, ops uint64) string {
	timeConsume := uint64(stopTime.Sub(startTime).Nanoseconds() / 1000)
	doCounts := ops * 1000 * 1000

	tps := uint64(doCounts / timeConsume)
	return fmt.Sprintf("%s: %d tps. %d microseconds(%f seconds).", t.option.name, tps, timeConsume, float64(timeConsume)/(1000*1000))
}

func (t *tps) Print() {
	if t.startTime == nil {
		log15.Crit("Must call `Start` function before call `Print` function", "method", "tps/Print")
	}
	if t.stopTime == nil {
		log15.Crit("Must call `Stop` function before call `Print` function", "method", "tps/Print")
	}
	fmt.Printf("[final statistical result] %s %d total ops.\n", t.print(*t.startTime, *t.stopTime, t.Ops()), t.Ops())

}
