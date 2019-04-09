package p2p

import (
	"testing"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

type mockPeer struct {
	id    vnode.NodeID
	level Level
}

func (mp *mockPeer) WriteMsg(Msg) error {
	panic("implement me")
}

func (mp *mockPeer) ID() vnode.NodeID {
	return mp.id
}

func (mp *mockPeer) String() string {
	panic("implement me")
}

func (mp *mockPeer) Address() string {
	panic("implement me")
}

func (mp *mockPeer) Info() PeerInfo {
	panic("implement me")
}

func (mp *mockPeer) Close(err PeerError) error {
	panic("implement me")
}

func (mp *mockPeer) Level() Level {
	return mp.level
}

func (mp *mockPeer) SetLevel(level Level) error {
	panic("implement me")
}

func (mp *mockPeer) run() error {
	panic("implement me")
}

func (mp *mockPeer) setManager(pm levelManager) {
	panic("implement me")
}

func TestPeers_add(t *testing.T) {
	// no slot
	t.Log("add peer when no slots")
	var ps = newPeers(map[Level]int{})
	peer := &mockPeer{
		level: Superior,
		id:    vnode.RandomNodeID(),
	}
	pe, ok := ps.add(peer)
	if ok {
		t.Fail()
	}
	if pe != PeerTooManyPeers {
		t.Fail()
	}

	// some low level has a slot
	t.Log("some low level has slots")
	ps = newPeers(map[Level]int{
		Inbound:  1,
		Outbound: 1,
	})

	pe, ok = ps.add(peer)
	if !ok {
		t.Fail()
	}
	if ps.levelCount(Outbound) != 1 {
		t.Error("wrong level count")
	}

	// add again
	t.Log("add the same peer again")
	pe, ok = ps.add(peer)
	if ok {
		t.Fail()
	}
	if pe != PeerAlreadyConnected {
		t.Fail()
	}

	peer.id = vnode.RandomNodeID()
	_, ok = ps.add(peer)
	if !ok {
		t.Error("should add success")
	}
	if ps.count() != 2 {
		t.Errorf("wrong count: %d", ps.count())
	}

	// low level has a slot
	t.Log("lowest level has slots")
	ps = newPeers(map[Level]int{
		Inbound: 1,
	})

	pe, ok = ps.add(peer)
	if !ok {
		t.Log("add failed")
		t.Fail()
	}
	if ps.count() != 1 {
		t.Log("count", ps.count())
		t.Fail()
	}
}

func TestPees_remove(t *testing.T) {
	var ps = newPeers(map[Level]int{
		Inbound:  1,
		Outbound: 1,
		Superior: 1,
	})

	peer := &mockPeer{
		level: Superior,
		id:    vnode.RandomNodeID(),
	}

	err := ps.remove(peer)
	if err != errPeerNotExist {
		t.Error("should remove error")
	}

	_, ok := ps.add(peer)
	if !ok {
		t.Error("should add success")
	}
	if ps.levelCount(Superior) != 1 {
		t.Error("wrong level count")
	}

	err = ps.remove(peer)
	if err != nil {
		t.Error("should remove success")
	}

	if ps.has(peer.id) {
		t.Error("should remove")
	}

	if ps.count() != 0 {
		t.Error("wrong count")
	}
}

func TestPeers_changeLevel(t *testing.T) {
	var ps = newPeers(map[Level]int{
		Inbound:  1,
		Outbound: 1,
		Superior: 1,
	})

	peer := &mockPeer{
		level: Superior,
		id:    vnode.RandomNodeID(),
	}

	_, _ = ps.add(peer)

	peer.level = Inbound
	err := ps.changeLevel(peer, Superior)
	if err != nil {
		t.Error(err)
	}

	if ps.levelCount(Inbound) != 1 {
		t.Errorf("wrong inbound count: %d", ps.levelCount(Inbound))
	}
	if ps.levelCount(Outbound) != 0 {
		t.Errorf("wrong outbound count: %d", ps.levelCount(Outbound))
	}
	if ps.levelCount(Superior) != 0 {
		t.Errorf("wrong superior count: %d", ps.levelCount(Superior))
	}

	ps.resize(Superior, 0)
	peer.level = Superior
	err = ps.changeLevel(peer, Inbound)
	if err != errLevelIsFull {
		t.Errorf("change level failed: %v", err)
	}
	if ps.levelCount(Inbound) != 1 {
		t.Errorf("wrong inbound count: %d", ps.levelCount(Inbound))
	}
	if ps.levelCount(Superior) != 0 {
		t.Errorf("wrong superior count: %d", ps.levelCount(Superior))
	}
}

func TestPeers_resize(t *testing.T) {
	var ps = newPeers(map[Level]int{
		Outbound: 1,
	})

	ps.resize(Inbound, 2)
	if ps.levels[Inbound].max != 2 {
		t.Error("wrong resize")
	}

	ps.resize(Outbound, 2)
	if ps.levels[Outbound].max != 2 {
		t.Error("wrong resize")
	}
}

func TestPeers_has(t *testing.T) {
	var ps = newPeers(map[Level]int{
		Outbound: 1,
	})

	peer := &mockPeer{
		level: Superior,
		id:    vnode.RandomNodeID(),
	}

	_, ok := ps.add(peer)
	if !ok {
		t.Fail()
	}

	if !ps.has(peer.id) {
		t.Fail()
	}
}
