package net

import (
	"testing"

	"github.com/vitelabs/go-vite/p2p"

	"time"

	"github.com/vitelabs/go-vite/common/types"
)

func TestFilter_Hold(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		t.Fatal(e)
	}

	f := newFilter()

	// first time, should not hold
	if _, hold := f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}

	// should exist in map
	r, ok := f.records[hash]
	if !ok {
		t.Fatal("record should exist")
	}

	// have`nt done, then hold 2*timeThreshold and maxMark*2 times
	for i := 0; i < maxMark*2; i++ {
		if _, hold := f.hold(hash); !hold {
			t.Fatal("should hold, because mark is enough, but time is too short")
		}
	}

	if r.mark != maxMark*2 {
		t.Fatalf("record mark should be maxMark*2, but get %d\n", r.mark)
	}

	// have wait exceed 2 * timeThreshold and maxMark times from add
	time.Sleep(2 * timeThreshold * time.Second)

	_, hold := f.hold(hash)
	if hold {
		t.Fatalf("should not hold, mark %d, elapse %d", r.mark, time.Now().Unix()-r.addAt)
	}
}

func TestFilter_Done(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}

	f := newFilter()

	var id p2p.MsgId
	var hold bool
	// first time, should hold
	if id, hold = f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}

	f.done(id)

	// task done, then hold maxMark times and timeThreshold
	for i := 0; i < maxMark-1; i++ {
		if _, hold = f.hold(hash); !hold {
			t.Fatal("should hold, because mark is enough, but time is too short")
		}
	}

	// have done, then hold timeThreshold
	time.Sleep(timeThreshold * time.Second)
	if _, hold = f.hold(hash); hold {
		t.Fatal("should not hold")
	}
}

func TestFilter_fail(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}

	f := newFilter()

	var id p2p.MsgId
	var hold bool
	// first time, should hold
	if id, hold = f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}

	f.fail(id)
	if _, hold = f.hold(hash); hold {
		t.Fatal("should not hold after fail")
	}
}

func TestFilter_clean(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}

	f := newFilter()

	var id p2p.MsgId
	var hold bool
	// first time, should hold
	if id, hold = f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}
	if f.idToHash[id] != hash {
		t.Fail()
	}
	if len(f.idToHash) != 1 {
		t.Fail()
	}
	if len(f.records) != 1 {
		t.Fail()
	}

	f.clean(time.Now().Add(5 * time.Minute).Unix())
	if len(f.idToHash) != 0 {
		t.Fail()
	}
	if len(f.records) != 0 {
		t.Fail()
	}
}

func TestPolicy_AccountTargets(t *testing.T) {
	peers := newPeerSet()
	f := &fp{
		peers: peers,
	}

	if f.account(0) != nil {
		t.Fail()
	}
}

func TestPolicy_SnapshotTarget(t *testing.T) {
	peers := newPeerSet()
	f := &fp{
		peers: peers,
	}

	if f.snapshot(0) != nil {
		t.Fail()
	}

	const total = 10
	for i := uint64(1); i < total; i++ {
		err := peers.add(&peer{
			//id:     strconv.FormatUint(i, 10),
			//height: i,
		})
		if err != nil {
			t.Fatal("add peer error")
		}
	}

	p := f.snapshot(0)
	if p.height() != total-1 {
		t.Log("should be best peer")
		t.Fail()
	}

	p = f.snapshot(total)
	if p != nil {
		t.Fail()
	}

	p = f.snapshot(total / 2)
	if p.height() < total/2 {
		t.Fail()
	}
}
