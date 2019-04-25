package net

import (
	"fmt"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestRecord_Fail(t *testing.T) {
	var r = &record{
		st:    reqPending,
		fails: make([]peerId, 0, keepFails),
	}

	var id peerId
	id = vnode.RandomNodeID()
	r.fail(id)
	if r.st != reqError {
		t.Errorf("wrong st: %d", r.st)
	}
	if r.t != time.Now().Unix() {
		t.Error("wrong time")
	}
	if len(r.fails) != 1 || r.fails[0] != id {
		t.Error("wrong fail id")
	}

	r.reset()
	for i := 0; i < keepFails; i++ {
		id = vnode.RandomNodeID()
		r.fail(id)
		r.st = reqPending
	}
	if len(r.fails) != keepFails {
		t.Errorf("wrong fails %d", len(r.fails))
	}

	id = vnode.RandomNodeID()
	r.fail(id)
	if len(r.fails) != 3 || r.fails[0] != id {
		t.Errorf("wrong recent fail id")
	}

	fmt.Println(r.fails)
}

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

	var r *record
	var hold bool
	// first time, should hold
	if r, hold = f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}

	f.done(r.id)

	// task done, then hold maxMark times and timeThreshold
	for i := 0; i < maxMark; i++ {
		if _, hold = f.hold(hash); false == hold {
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

	var r *record
	var hold bool
	// first time, should hold
	if r, hold = f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}

	f.failNoPeers(r.id)
	if _, hold = f.hold(hash); hold {
		t.Fatal("should not hold after fail")
	}

	for i := 0; i < keepFails; i++ {
		r.st = reqPending
		id := vnode.RandomNodeID()
		r.fail(id)
	}
	if _, hold = f.hold(hash); false == hold {
		t.Fatal("should hold, because fail too short time")
	}
	time.Sleep(fail3wait * time.Second)
	if _, hold = f.hold(hash); true == hold {
		t.Fatal("should not hold, because fail too long time")
	}
}

func TestFilter_clean(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}

	f := newFilter()

	var r *record
	var hold bool
	// first time, should hold
	if r, hold = f.hold(hash); hold {
		t.Fatal("should not hold first time")
	}
	if f.idToHash[r.id] != hash {
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
	var err error
	var chooseTop = true
	m := newPeerSet()
	f := &fetchTarget{
		peers: m,
		chooseTop: func() bool {
			return chooseTop
		},
	}

	var r = &record{
		st:    reqPending,
		fails: make([]peerId, 0, keepFails),
	}

	var ret Peer

	// no peers
	if ret = f.account(0, r); ret != nil {
		t.Error("peer should be nil")
	}

	// only one peer
	ret = newMockPeer(vnode.RandomNodeID(), 1)
	if err = m.add(ret); err != nil {
		t.Errorf("failed to add peer: %v", err)
	}
	if ret = f.account(10, r); ret == nil {
		t.Error("account target should not be nil")
	}

	// two peers
	if err = m.add(newMockPeer(vnode.RandomNodeID(), 2)); err != nil {
		t.Errorf("failed to add peer: %v", err)
	}

	if ret = f.account(1, r); ret == nil {
		t.Error("failed to get account target")
		return
	}

	// fail one
	r.fail(ret.ID())
	if ret = f.account(1, r); ret == nil {
		t.Error("failed to get account target")
	}
	// another is too low
	if ret = f.account(20, r); ret != nil {
		t.Error("account should be nil, because height is too low")
	}

	// ten peers [1 ... 10]
	for i := 3; i < 11; i++ {
		if err = m.add(newMockPeer(vnode.RandomNodeID(), uint64(i))); err != nil {
			t.Errorf("failed to add peer: %v", err)
		}
	}

	// choose from top: 10 9 8
	for i := 0; i < 100; i++ {
		if ret = f.account(0, r); ret.height() < 8 {
			t.Errorf("not top peers: %d", ret.height())
		}
	}

	// 10 9 8 failed
	ps := m.sortPeers()
	for i := 0; i < keepFails; i++ {
		r.st = reqPending
		r.fail(ps[i].ID())
	}

	for i := 0; i < 100; i++ {
		if ret = f.account(0, r); ret.height() > 7 {
			t.Errorf("wrong top peers: %d", ret.height())
		}
	}
}

func TestPolicy_SnapshotTarget(t *testing.T) {
	var err error
	var ps peers

	m := newPeerSet()
	f := &fetchTarget{
		peers: m,
	}

	var r = &record{
		st:    reqPending,
		fails: make([]peerId, 0, keepFails),
	}

	if f.snapshot(0, r) != nil {
		t.Fail()
	}

	const total = 10
	for i := 1; i < total; i++ {
		if err = m.add(newMockPeer(vnode.RandomNodeID(), uint64(i))); err != nil {
			t.Fatal("add peer error")
		}
	}

	var ret Peer
	if ret = f.snapshot(20, r); ret != nil {
		t.Errorf("should be nil, because peers are too low")
		return
	}

	ps = m.sortPeers()
	if ret = f.snapshot(0, r); ret != ps[0] {
		t.Errorf("should be the best peer")
	}

	for i := 0; i < keepFails; i++ {
		ret = ps[i]
		fmt.Println("fail", ret.ID(), ret.height())
		r.fail(ret.ID())
		r.st = reqPending
	}
	fmt.Println(r.fails)

	if ret = f.snapshot(0, r); ret != ps[keepFails] {
		t.Errorf("should be the %d peer", keepFails)
	}
}
