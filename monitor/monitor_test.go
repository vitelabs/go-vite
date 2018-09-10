package monitor

import (
	"testing"
	"time"
)

func TestMonitor(t *testing.T) {
	// record the number of times event happened
	LogEvent("pool", "insert")
	// record the time taken by sth
	testLogTime1()
	testLogTime2()
	// record average
	LogDuration("pool", "insertDuration", 2000)
}

func testLogTime2() {
	last := time.Now()
	time.Sleep(time.Second)
	LogTime("pool", "insertTime2", last)
}

func testLogTime1() {
	defer LogTime("pool", "insertTime1", time.Now())
	time.Sleep(time.Second)
}
