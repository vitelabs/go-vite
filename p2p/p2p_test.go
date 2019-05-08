package p2p

import (
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	var initDuration = 1 * time.Second
	var maxDuration = 4 * time.Second
	var duration = initDuration
	var timer = time.NewTimer(duration)
	defer timer.Stop()

	var start = time.Now().Unix()
	var i int
	for {
		i++

		if i > 3 {
			break
		}
		<-timer.C

		if duration < maxDuration {
			duration *= 2
		} else {
			duration = initDuration
		}

		timer.Reset(duration)
	}

	var stop = time.Now().Unix()
	if stop-start != 7 {
		t.Errorf("error duration: %d", stop-start)
	}
}

//var blockUtil = block.New(blockPolicy)
//
//func TestBlock(t *testing.T) {
//	var id discovery.NodeID
//	rand.Read(id[:])
//
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	blockUtil.Block(id[:])
//
//	if !blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	time.Sleep(blockMinExpired)
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//}
//
//func TestBlock_F(t *testing.T) {
//	var id discovery.NodeID
//	rand.Read(id[:])
//
//	for i := 0; i < blockCount-1; i++ {
//		blockUtil.Block(id[:])
//	}
//
//	if !blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	time.Sleep(blockMinExpired)
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	blockUtil.Block(id[:])
//
//	time.Sleep(blockMinExpired)
//	if !blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	time.Sleep(blockMaxExpired)
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//}
