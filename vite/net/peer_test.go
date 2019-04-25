package net

import (
	"fmt"
	"math/rand"
	net2 "net"
	"sort"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

type mockPeer struct {
	id      vnode.NodeID
	_height uint64
	peerMap map[vnode.NodeID]struct{}
}

func newMockPeer(id vnode.NodeID, height uint64) *mockPeer {
	return &mockPeer{
		id:      id,
		_height: height,
		peerMap: make(map[vnode.NodeID]struct{}),
	}
}

func (mp *mockPeer) WriteMsg(p2p.Msg) error {
	return nil
}

func (mp *mockPeer) ID() vnode.NodeID {
	return mp.id
}

func (mp *mockPeer) String() string {
	return mp.id.Brief()
}

func (mp *mockPeer) Address() net2.Addr {
	return &net2.TCPAddr{
		IP:   []byte{127, 0, 0, 1},
		Port: 0,
		Zone: "mock",
	}
}

func (mp *mockPeer) Info() p2p.PeerInfo {
	return p2p.PeerInfo{
		ID:        mp.id.Brief(),
		Name:      "",
		Version:   0,
		Protocols: nil,
		Address:   "",
		Level:     0,
		CreateAt:  "",
		State:     nil,
	}
}

func (mp *mockPeer) Close(err p2p.PeerError) error {
	return nil
}

func (mp *mockPeer) Level() p2p.Level {
	panic("implement me")
}

func (mp *mockPeer) SetLevel(level p2p.Level) error {
	panic("implement me")
}

func (mp *mockPeer) State() interface{} {
	panic("implement me")
}

func (mp *mockPeer) SetState(state interface{}) {
	panic("implement me")
}

func (mp *mockPeer) setHead(head types.Hash, height uint64) {
	panic("implement me")
}

func (mp *mockPeer) setPeers(ps []peerConn, patch bool) {
	var id vnode.NodeID
	var err error

	if patch {
		mp.peerMap = make(map[vnode.NodeID]struct{})
	}

	for _, c := range ps {
		if c.add {
			id, err = vnode.Bytes2NodeID(c.id)
			if err != nil {
				continue
			}
			mp.peerMap[id] = struct{}{}
		} else {
			delete(mp.peerMap, id)
		}
	}
}

func (mp *mockPeer) peers() map[vnode.NodeID]struct{} {
	return mp.peerMap
}

func (mp *mockPeer) seeBlock(hash types.Hash) bool {
	panic("implement me")
}

func (mp *mockPeer) height() uint64 {
	return mp._height
}

func (mp *mockPeer) head() types.Hash {
	panic("implement me")
}

func (mp *mockPeer) fileAddress() string {
	panic("implement me")
}

func (mp *mockPeer) catch(err error) {
	panic("implement me")
}

func (mp *mockPeer) send(c code, id p2p.MsgId, data p2p.Serializable) error {
	panic("implement me")
}

func (mp *mockPeer) sendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId p2p.MsgId) error {
	panic("implement me")
}

func (mp *mockPeer) sendAccountBlocks(bs []*ledger.AccountBlock, msgId p2p.MsgId) error {
	panic("implement me")
}

func (mp *mockPeer) info() PeerInfo {
	panic("implement me")
}

func TestPeerSet_Add(t *testing.T) {
	var m = newPeerSet()
	var p = newMockPeer(vnode.RandomNodeID(), 1)

	// add first time
	if m.add(p) != nil {
		t.Fail()
	}

	// add repeatedly
	if m.add(p) == nil {
		t.Fail()
	}
}

func TestPeerSet_Del(t *testing.T) {
	var m = newPeerSet()
	var p = newMockPeer(vnode.RandomNodeID(), 1)

	// should have no error
	if m.add(p) != nil {
		t.Fail()
	}

	p2, err := m.remove(p.ID())
	if err != nil {
		t.Fail()
	}

	if p != p2 {
		t.Fail()
	}

	if p2 = m.get(p.ID()); p2 != nil {
		t.Fail()
	}
}

func TestPeerSet_SyncPeer(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	var err error

	if m.syncPeer() != nil {
		t.Fail()
	}

	if m.bestPeer() != nil {
		t.Fail()
	}

	for i := 0; i < 2; i++ {
		p = newMockPeer(vnode.RandomNodeID(), uint64(i))
		if err = m.add(p); err != nil {
			t.Errorf("failed to add peer: %v", err)
		}
	}

	// only two peers
	if p = m.syncPeer(); p.height() != 1 {
		t.Errorf("wrong sync peer height: %d", p.height())
	}

	// reset
	m = newPeerSet()

	const total = 10
	for i := 0; i < total; i++ {
		p = newMockPeer(vnode.RandomNodeID(), uint64(i))
		if m.add(p) != nil {
			t.Fail()
		}
	}

	// the 1/3 peer
	if p = m.syncPeer(); p.height() != 6 {
		t.Errorf("wrong sync peer height: %d", p.height())
	}
	if p = m.bestPeer(); p.height() != total-1 {
		t.Errorf("wrong best peer height: %d", p.height())
	}
}

func TestPeerSet_Pick(t *testing.T) {
	var m = newPeerSet()
	var p Peer

	const total = 10
	for i := 0; i < total; i++ {
		p = newMockPeer(vnode.RandomNodeID(), uint64(i))
		if m.add(p) != nil {
			t.Fail()
		}
	}

	const target = total / 2

	ps := m.pick(target)
	if len(ps) != total/2 {
		t.Errorf("wrong peer number: %d", len(ps))
	}
	for _, p = range ps {
		if p.height() < target {
			t.Errorf("wrong peer height: %d", p.height())
		}
	}
}

func ExamplePeerSet_Get() {
	var m1 = newPeerSet()
	var p1 = m1.get(vnode.ZERO)

	var m2 = make(map[string]Peer)
	var p2 = m2["hello"]

	var m3 = make(map[string]*peer)
	var p3 Peer = m3["hello"]

	fmt.Println(p1 == nil)
	fmt.Println(p2 == nil)
	fmt.Println(p3 == nil)
	// Output:
	// true
	// true
	// false
}

func ExamplePeersSort() {
	var ps peers
	ps = append(ps, newMockPeer(vnode.RandomNodeID(), 1))
	ps = append(ps, newMockPeer(vnode.RandomNodeID(), 2))
	ps = append(ps, newMockPeer(vnode.RandomNodeID(), 3))

	sort.Sort(ps)
	fmt.Println(ps[0].height())
	// Output:
	// 3
}

// test read write concurrently
func TestPeer_peers(t *testing.T) {
	var p = &peer{
		m: make(map[peerId]struct{}),
	}

	go func() {
		patch := false
		const batch = 10
		for {
			ps := make([]peerConn, batch)
			for i := range ps {
				ps[i] = peerConn{
					id:  vnode.RandomNodeID().Bytes(),
					add: true,
				}
			}
			p.setPeers(ps, patch)
			patch = !patch
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		for {
			mp := p.peers()
			for k := range mp {
				delete(mp, k)
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	time.Sleep(5 * time.Second)
}

func TestPeer_peers2(t *testing.T) {
	const maxNeighbors = 200
	var total = rand.Intn(maxNeighbors)

	var pids = make([]vnode.NodeID, total)
	for i := range pids {
		pids[i] = vnode.RandomNodeID()
	}

	var m, m2 map[vnode.NodeID]struct{}

	m = make(map[vnode.NodeID]struct{})
	for i := 0; i < total/2; i++ {
		m[pids[i]] = struct{}{}
	}

	var p = &peer{
		m: m,
	}

	p.setPeers(nil, true)
	m2 = p.peers()
	if len(m2) != len(p.m) {
		t.Errorf("wrong peers: %d", len(m2))
	}

	p.setPeers(nil, false)
	m2 = p.peers()
	if len(m2) != 0 {
		t.Errorf("wrong peers: %d", len(m2))
	}
	// reset
	p.m = m
	p.m2 = m

	idsPatch := make([]peerConn, total/4)
	var add = true
	for i := 0; i < len(idsPatch); i++ {
		idsPatch[i] = peerConn{
			id:  pids[i].Bytes(),
			add: add,
		}
		add = !add
	}

	old := len(m)
	p.setPeers(idsPatch, true)
	m2 = p.peers()
	should := old - len(idsPatch)/2

	if len(m2) != should {
		t.Errorf("should %d peers, but %d peers", should, len(m2))
	}
}
