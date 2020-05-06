package monitor

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLogTime(t *testing.T) {
	go func() {
		for {
			LogTime("a", "b", time.Now().Add(-time.Second*30))
			time.Sleep(200 * time.Millisecond)
		}
	}()

	go func() {
		t := time.NewTicker(time.Millisecond * 500)
		for {
			select {
			case <-t.C:
				println(StatsJson())
			}
		}
	}()

	//time.Sleep(10 * time.Second)
	//go func() {
	//	for {
	//		LogTime("a", "b", time.Now().Add(-time.Second*60))
	//		time.Sleep(200 * time.Millisecond)
	//	}
	//}()

	<-make(chan int)

}

func TestJson(t *testing.T) {
	add := newMsg("a", "b").add(1).add(2)
	bytes, _ := json.Marshal(add)
	t.Log(string(bytes))
	if string(bytes) != add.String() {
		t.Error("json format fail.")
	}
}

func TestSnapshot(t *testing.T) {
	a := newMsg("a", "b").add(1).add(2)
	snapshot := a.snapshot()

	a.add(1)

	println((&snapshot).String())
	println(a.String())
	if (&snapshot).String() == a.String() {
		t.Error("snapshot fail.")
	}
}
