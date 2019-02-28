package net

import (
	"strconv"
	"testing"

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
	if f.hold(hash) {
		t.Fatal("should not hold first time")
	}

	// should exist in map
	r, ok := f.records[hash]
	if !ok {
		t.Fatal("record should exist")
	}

	// have`nt done, then hold 2*timeThreshold and maxMark*2 times
	for i := 0; i < maxMark*2-1; i++ {
		if !f.hold(hash) {
			t.Fatal("should hold, because mark is enough, but time is too short")
		}
	}

	if r.mark != maxMark*2 {
		t.Fatalf("record mark should be maxMark*2, but get %d\n", r.mark)
	}

	// have wait exceed 2 * timeThreshold and maxMark times from add
	time.Sleep(2 * timeThreshold * time.Second)

	hold := f.hold(hash)
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

	// first time, should hold
	if f.hold(hash) {
		t.Fatal("should not hold first time")
	}

	f.done(hash)

	// task done, then hold maxMark times and timeThreshold
	for i := 0; i < maxMark-1; i++ {
		if !f.hold(hash) {
			t.Fatal("should hold, because mark is enough, but time is too short")
		}
	}

	// have done, then hold timeThreshold
	time.Sleep(timeThreshold * time.Second)
	if f.hold(hash) {
		t.Fatal("should not hold")
	}
}

func TestPolicy_AccountTargets(t *testing.T) {
	peers := newPeerSet()
	f := &fp{
		peers: peers,
	}

	if f.accountTargets(0) != nil {
		t.Fail()
	}

	i := uint64(1)
	err := peers.Add(&peer{
		id:     strconv.FormatUint(i, 10),
		height: i,
	})
	if err != nil {
		t.Fatal("add peer error")
	}

	l := f.accountTargets(0)
	if len(l) != 1 {
		t.Logf("should get 1 account targets, but get %d\n", len(l))
		t.Fail()
	}

	for i = uint64(2); i < 10; i++ {
		err = peers.Add(&peer{
			id:     strconv.FormatUint(i, 10),
			height: i,
		})
		if err != nil {
			t.Fatal("add peer error")
		}
	}

	l = f.accountTargets(10)
	if len(l) != 2 {
		t.Logf("should get 2 account targets, but get %d\n", len(l))
		t.Fail()
	}
}

func TestPolicy_SnapshotTarget(t *testing.T) {
	peers := newPeerSet()
	f := &fp{
		peers: peers,
	}

	if f.snapshotTarget(0) != nil {
		t.Fail()
	}

	const total = 10
	for i := uint64(1); i < total; i++ {
		err := peers.Add(&peer{
			id:     strconv.FormatUint(i, 10),
			height: i,
		})
		if err != nil {
			t.Fatal("add peer error")
		}
	}

	p := f.snapshotTarget(0)
	if p.Height() != total-1 {
		t.Log("should be best peer")
		t.Fail()
	}

	p = f.snapshotTarget(total)
	if p != nil {
		t.Fail()
	}

	p = f.snapshotTarget(total / 2)
	if p.Height() < total/2 {
		t.Fail()
	}
}
