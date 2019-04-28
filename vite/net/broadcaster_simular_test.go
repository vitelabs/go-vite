package net

import (
	"errors"
	"fmt"
	net2 "net"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/vite/net/message"
)

type mockBroadcastPeerSet map[vnode.NodeID]*mockBroadcastPeer

func (m mockBroadcastPeerSet) broadcastPeers() (l []broadcastPeer) {
	for _, p := range m {
		l = append(l, p)
	}

	return
}

type mockVerifier struct{}

func (mockVerifier) VerifyNetSb(block *ledger.SnapshotBlock) error {
	return nil
}

func (mockVerifier) VerifyNetAb(block *ledger.AccountBlock) error {
	return nil
}

type mockBlockNotifier struct{}

func (mockBlockNotifier) notifySnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) {
}

func (mockBlockNotifier) notifyAccountBlock(block *ledger.AccountBlock, source types.BlockSource) {
}

type blockMessage struct {
	p2p.Msg
	sender vnode.NodeID
}

type mockBroadcastNet struct {
	*broadcaster
	id        vnode.NodeID
	priv      ed25519.PrivateKey
	rw        sync.RWMutex
	peerSet   mockBroadcastPeerSet
	transport chan blockMessage
	term      chan struct{}
}

type mockBroadcastPeer struct {
	fromId vnode.NodeID
	*mockBroadcastNet
	knownBlocks map[types.Hash]struct{}
	rw          sync.RWMutex
}

func (mpf *mockBroadcastPeer) seeBlock(hash types.Hash) bool {
	mpf.rw.Lock()
	defer mpf.rw.Unlock()

	if _, ok := mpf.knownBlocks[hash]; ok {
		return ok
	}

	mpf.knownBlocks[hash] = struct{}{}
	return false
}

func (mpf *mockBroadcastPeer) sendNewSnapshotBlock(block *ledger.SnapshotBlock) error {
	mpf.seeBlock(block.Hash)

	var msg = &message.NewSnapshotBlock{
		Block: block,
		TTL:   10,
	}
	buf, err := msg.Serialize()
	if err != nil {
		panic(err)
	}

	select {
	case mpf.transport <- blockMessage{
		Msg: p2p.Msg{
			Code:    p2p.Code(NewSnapshotBlockCode),
			Payload: buf,
		},
		sender: mpf.fromId,
	}:
	case <-mpf.term:
		return errors.New("peer stop")
	}

	return nil
}

func (mpf *mockBroadcastPeer) sendNewAccountBlock(block *ledger.AccountBlock) error {
	mpf.seeBlock(block.Hash)

	var msg = &message.NewAccountBlock{
		Block: block,
		TTL:   10,
	}
	buf, err := msg.Serialize()
	if err != nil {
		panic(err)
	}

	select {
	case mpf.transport <- blockMessage{
		Msg: p2p.Msg{
			Code:    p2p.Code(NewAccountBlockCode),
			Payload: buf,
		},
		sender: mpf.fromId,
	}:
	case <-mpf.term:
		return errors.New("peer stop")
	}

	return nil
}

func (mpf *mockBroadcastPeer) send(c code, id p2p.MsgId, data p2p.Serializable) error {
	buf, err := data.Serialize()
	if err != nil {
		panic(err)
	}

	if a, ok := data.(*message.NewAccountBlock); ok {
		mpf.seeBlock(a.Block.Hash)
		select {
		case mpf.transport <- blockMessage{
			Msg: p2p.Msg{
				Code:    p2p.Code(c),
				Id:      id,
				Payload: buf,
			},
			sender: mpf.fromId,
		}:
		case <-mpf.term:
			return errors.New("peer stop")
		}
	} else if b, ok := data.(*message.NewSnapshotBlock); ok {
		mpf.seeBlock(b.Block.Hash)
		select {
		case mpf.transport <- blockMessage{
			Msg: p2p.Msg{
				Code:    p2p.Code(c),
				Id:      id,
				Payload: buf,
			},
			sender: mpf.fromId,
		}:
		case <-mpf.term:
			return errors.New("peer stop")
		}
	} else {
		return errors.New("message should be new block")
	}

	return nil
}

func (mp *mockBroadcastNet) loop(wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	//var senderId vnode.NodeID
	//var sender broadcastPeer

	for {
		select {
		case <-mp.term:
			return
			//case msg := <-mp.transport:
			//senderId = msg.sender

			//mp.rw.RLock()
			//sender = mp.peerSet[senderId]
			//mp.rw.RUnlock()

			//go mp.broadcaster.Handle(msg.Msg, sender)
		}
	}
}

func (mp *mockBroadcastNet) stop() {
	close(mp.term)
}

type mockNewBlockListener struct {
	r  map[types.Hash]int
	mu sync.Mutex
}

func (m *mockNewBlockListener) onNewAccountBlock(block *ledger.AccountBlock) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.r[block.Hash] = m.r[block.Hash] + 1
}

func (m *mockNewBlockListener) onNewSnapshotBlock(block *ledger.SnapshotBlock) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.r[block.Hash] = m.r[block.Hash] + 1
}

var globalMockNewBlockListener = &mockNewBlockListener{
	r: make(map[types.Hash]int),
}

func newMockBroadcastNet(id vnode.NodeID, priv ed25519.PrivateKey) *mockBroadcastNet {
	var mp = &mockBroadcastNet{
		broadcaster: nil,
		id:          id,
		priv:        priv,
		rw:          sync.RWMutex{},
		peerSet:     nil,
		transport:   make(chan blockMessage, 3),
		term:        make(chan struct{}),
	}

	// create peerFrames
	peerFrames := make(map[vnode.NodeID]*mockBroadcastPeer)

	set := mockBroadcastPeerSet(peerFrames)
	mp.peerSet = set

	str := newCrossForwardStrategy(set, 1, 10)
	//str := &fullForwardStrategy{
	//	ps: set,
	//}

	b := newBroadcaster(set, mockVerifier{}, mockBlockNotifier{}, nil, str, globalMockNewBlockListener, nil)
	b.st = SyncDone

	mp.broadcaster = b

	return mp
}

func (mp *mockBroadcastNet) addPeer(peer *mockBroadcastNet) int {
	if _, ok := mp.peerSet[peer.id]; ok {
		return len(mp.peerSet)
	}

	mp.peerSet[peer.id] = &mockBroadcastPeer{
		fromId:           mp.id,
		mockBroadcastNet: peer,
		knownBlocks:      make(map[types.Hash]struct{}),
	}

	peer.addPeer(mp)

	return len(mp.peerSet)
}

func (mp *mockBroadcastNet) ID() vnode.NodeID {
	return mp.id
}

type mpAddr struct {
	str string
}

func (m mpAddr) Network() string {
	return "mock broadcast"
}

func (m mpAddr) String() string {
	return m.Network() + " " + m.str
}

func (mp *mockBroadcastNet) Address() net2.Addr {
	return mpAddr{
		str: mp.id.String(),
	}
}

func (mp *mockBroadcastNet) peers() map[vnode.NodeID]struct{} {
	mp.rw.RLock()
	defer mp.rw.RUnlock()

	m := make(map[vnode.NodeID]struct{})
	for id := range mp.peerSet {
		m[id] = struct{}{}
	}

	return m
}

func (mp *mockBroadcastNet) catch(error) {
	return
}

func TestBroadcaster_Reduce(t *testing.T) {
	const total = 30
	const avgPeers = 30

	// generate all peers
	var allNets = make(map[vnode.NodeID]*mockBroadcastNet)
	for i := 0; i < total; i++ {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			panic(err)
		}

		id, _ := vnode.Bytes2NodeID(pub)
		allNets[id] = newMockBroadcastNet(id, priv)
	}

	var initiator *mockBroadcastNet

	// random neighbors
	for id, p := range allNets {
		if initiator == nil {
			initiator = p
		}

		for id2, p2 := range allNets {
			if id == id2 {
				continue
			}

			if p.addPeer(p2) > avgPeers {
				break
			}
		}
	}

	var wg sync.WaitGroup
	// start all nets
	for _, p := range allNets {
		go p.loop(&wg)
	}

	var prevHash types.Hash
	stateHash, _ := types.HexToHash("cca5fc60c1d1e103127952fffef0994a6e7b3d89310a1423de7ae223ec639bab")
	timestamp := time.Unix(1550578308, 0)

	addr1, _ := types.HexToAddress("vite_00000000000000000000000000000000000000056ad6d26692")
	addr1Hash, _ := types.HexToHash("41de9e174e848bea771ffe6386afc47dd2384ffb952e27bc27122d257a7bf7c5")

	addr2, _ := types.HexToAddress("vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23")
	addr2Hash, _ := types.HexToHash("acd6512bcc2ba80bc6205763d1214942b75263d83a7631f5c2a22378b4878817")

	for i := uint64(0); i < 1; i++ {
		block := &ledger.SnapshotBlock{
			Hash:      types.Hash{},
			PrevHash:  prevHash,
			Height:    i,
			PublicKey: initiator.id[:],
			Signature: nil,
			Timestamp: &timestamp,
			Seed:      0,
			SeedHash:  &stateHash,
			SnapshotContent: ledger.SnapshotContent{
				addr1: {
					Height: 15,
					Hash:   addr1Hash,
				},
				addr2: {
					Height: 534,
					Hash:   addr2Hash,
				},
			},
		}

		block.Hash = block.ComputeHash()
		block.Signature = ed25519.Sign(initiator.priv, block.Hash[:])

		initiator.BroadcastSnapshotBlock(block)

		prevHash = block.Hash
	}

	for _, n := range allNets {
		n.stop()
	}

	wg.Wait()

	for id, count := range globalMockNewBlockListener.r {
		fmt.Println(id.String(), count)
	}
}
