package net

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/vite/net/circle"
)

func TestBroadcaster_Statistic(t *testing.T) {
	bdc := &broadcaster{
		statistic: circle.NewList(records24h),
	}

	ret := bdc.Statistic()
	for _, v := range ret {
		if v != 0 {
			t.Fail()
		}
	}

	// put one element
	fmt.Println("put one elements")
	bdc.statistic.Put(int64(10))
	ret = bdc.Statistic()
	if ret[0] != 10 {
		t.Fail()
	}

	// put 3600 elements
	fmt.Println("put 3610 elements")
	bdc.statistic.Reset()
	var t0 int64
	var t1 float64
	var total = records1h + 10
	for i := 0; i < total; i++ {
		bdc.statistic.Put(int64(i))
	}
	ret = bdc.Statistic()
	for i := total - records1h; i < total; i++ {
		t1 += float64(i) / float64(records1h)
		t0 = int64(i)
	}
	fmt.Println(ret, t0, t1)
	if ret[0] != t0 {
		t.Fail()
	}
	if ret[1] != int64(t1) {
		t.Fail()
	}

	// put 43200 elements
	fmt.Println("put 43210 elements")
	bdc.statistic.Reset()
	t0, t1 = 0, 0

	var t12 float64
	total = records12h + 10
	for i := 0; i < total; i++ {
		bdc.statistic.Put(int64(i))
		t0 = int64(i)
	}
	ret = bdc.Statistic()
	for i := total - records1h; i < total; i++ {
		t1 += float64(i) / float64(records1h)
	}
	for i := total - records12h; i < total; i++ {
		t12 += float64(i) / float64(records12h)
	}
	fmt.Println(ret, t0, t1, t12)
	if ret[0] != t0 {
		t.Fail()
	}
	if ret[1] != int64(t1) {
		t.Fail()
	}
	if ret[2] != int64(t12) {
		t.Fail()
	}

	// put 86400 elements
	fmt.Println("put 86410 elements")
	bdc.statistic.Reset()
	t0, t1, t12 = 0, 0, 0

	var t24 float64
	total = records24h + 10
	for i := 0; i < total; i++ {
		bdc.statistic.Put(int64(i))
		t0 = int64(i)
	}
	ret = bdc.Statistic()
	for i := total - records1h; i < total; i++ {
		t1 += float64(i) / float64(records1h)
	}
	for i := total - records12h; i < total; i++ {
		t12 += float64(i) / float64(records12h)
	}
	for i := total - records24h; i < total; i++ {
		t24 += float64(i) / float64(records24h)
	}

	fmt.Println(ret, t0, t1, t12, t24)
	if ret[0] != t0 {
		t.Fail()
	}
	if ret[1] != int64(t1) {
		t.Fail()
	}
	if ret[2] != int64(t12) {
		t.Fail()
	}
	if ret[3] != int64(t24) {
		t.Fail()
	}
}

func BenchmarkBroadcaster_Statistic(b *testing.B) {
	bdc := &broadcaster{
		statistic: circle.NewList(records24h),
	}

	for i := int64(0); i < records24h*2; i++ {
		bdc.statistic.Put(i)
	}

	var ret []int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret = bdc.Statistic()
	}
	fmt.Println(ret)
}

func TestMemStore_EnqueueAccountBlock(t *testing.T) {
	store := newMemBlockStore(1000)
	for i := uint64(0); i < 1100; i++ {
		store.enqueueAccountBlock(&ledger.AccountBlock{
			Height: i,
		})
	}
	for i := uint64(0); i < 1000; i++ {
		block := store.dequeueAccountBlock()
		if block.Height != i {
			t.Fail()
		}
	}
	for i := uint64(0); i < 100; i++ {
		block := store.dequeueAccountBlock()
		if block != nil {
			t.Fail()
		}
	}
}

func TestMemStore_EnqueueSnapshotBlock(t *testing.T) {
	store := newMemBlockStore(1000)
	for i := uint64(0); i < 1100; i++ {
		store.enqueueSnapshotBlock(&ledger.SnapshotBlock{
			Height: i,
		})
	}
	for i := uint64(0); i < 1000; i++ {
		block := store.dequeueSnapshotBlock()
		if block.Height != i {
			t.Fail()
		}
	}
	for i := uint64(0); i < 100; i++ {
		block := store.dequeueSnapshotBlock()
		if block != nil {
			t.Fail()
		}
	}
}

//func BenchmarkBroadcaster_handle(b *testing.B) {
//	ps := newPeerSet()
//	feed := newBlockFeeder()
//	forward := createForardStrategy("cross", ps)
//	broadcaster := newBroadcaster(ps, &mockVerifier{}, feed, newMemBlockStore(1000), forward, nil, nil)
//
//	broadcaster.handle()
//}

func TestAccountMsgPool(t *testing.T) {
	p := newAccountMsgPool()

	for i := 0; i < 1000; i++ {
		var nb = new(message.NewAccountBlock)
		nb.Block = &ledger.AccountBlock{
			BlockType:      0,
			Hash:           types.Hash{byte(i % 256)},
			Height:         uint64(i),
			PrevHash:       types.Hash{},
			AccountAddress: types.Address{},
			PublicKey:      nil,
			ToAddress:      types.Address{},
			FromBlockHash:  types.Hash{},
			Amount:         new(big.Int),
			TokenId:        types.TokenTypeId{},
			Quota:          0,
			Fee:            new(big.Int),
			Data:           nil,
			LogHash:        nil,
			Difficulty:     nil,
			Nonce:          nil,
			Signature:      nil,
		}
		nb.TTL = int32(i)

		data, err := nb.Serialize()
		if err != nil {
			panic(err)
		}

		msg := p.get()
		err = msg.Deserialize(data)
		if err != nil {
			panic(err)
		}

		if msg.Block.Hash != nb.Block.Hash {
			t.Errorf("wrong hash")
		}
		if msg.TTL != nb.TTL {
			t.Errorf("wrong ttl")
		}

		p.put(msg)
	}
}

func TestSnapshotMsgPool(t *testing.T) {
	p := newSnapshotMsgPool()

	var now = time.Now()
	for i := 0; i < 1000; i++ {
		var nb = new(message.NewSnapshotBlock)
		nb.Block = &ledger.SnapshotBlock{
			Hash:            types.Hash{byte(i % 256)},
			PrevHash:        types.Hash{},
			Height:          uint64(i),
			PublicKey:       []byte("hello"),
			Signature:       []byte("hello"),
			Seed:            0,
			Timestamp:       &now,
			SeedHash:        &types.Hash{},
			SnapshotContent: nil,
		}
		nb.TTL = int32(i)

		data, err := nb.Serialize()
		if err != nil {
			panic(err)
		}

		msg := p.get()
		err = msg.Deserialize(data)
		if err != nil {
			panic(err)
		}

		if msg.Block.Hash != nb.Block.Hash {
			t.Errorf("wrong hash")
		}
		if msg.TTL != nb.TTL {
			t.Errorf("wrong ttl")
		}

		p.put(msg)
	}
}

func BenchmarkCrossPeers(b *testing.B) {
	var p = newMockPeer(vnode.RandomNodeID(), 1)
	const total = 200
	pcs := make([]peerConn, total)
	for i := 0; i < total; i++ {
		pcs[i] = peerConn{
			id:  vnode.RandomNodeID().Bytes(),
			add: true,
		}
	}
	p.setPeers(pcs, false)

	var ps = newPeerSet()
	for i := 0; i < 10; i++ {
		err := ps.add(newMockPeer(vnode.RandomNodeID(), 1))
		if err != nil {
			panic(err)
		}
	}
	cross := newCrossForwardStrategy(ps, 3, 10)

	b.ResetTimer()

	// 35000ns
	for i := 0; i < b.N; i++ {
		cross.choosePeers(p)
	}
}

func TestCommonPeers(t *testing.T) {
	const our = 100
	const common = 10
	var commonMax = 5
	var commonRatio = 100
	var ourPeers = make([]broadcastPeer, our)
	var ppMap map[peerId]struct{}
	var ps []broadcastPeer

	for i := 0; i < our; i++ {
		ourPeers[i] = newMockPeer(vnode.RandomNodeID(), 1)
	}

	copyOurPeers := func(n int) (l []broadcastPeer) {
		l = make([]broadcastPeer, n)
		for i := 0; i < n; i++ {
			l[i] = ourPeers[i]
		}

		return
	}

	verify := func(l []broadcastPeer) bool {
		for i, p := range l {
			if p == nil {
				t.Errorf("%d peer is nil", i)
				return false
			}
		}
		return true
	}

	sender := ourPeers[0].ID()

	// ppMap is nil
	ps = commonPeers(copyOurPeers(3), ppMap, sender, commonMax, commonRatio)
	if len(ps) != 2 {
		t.Errorf("wrong peers count %d", len(ps))
	} else if false == verify(ps) {
		t.Fail()
	}

	ppMap = make(map[peerId]struct{})
	for i := 0; i < common; i++ {
		ppMap[ourPeers[50+i].ID()] = struct{}{}
	}

	ps = commonPeers(copyOurPeers(3), ppMap, sender, commonMax, commonRatio)
	if len(ps) != 2 {
		t.Errorf("wrong peers count %d", len(ps))
	} else if false == verify(ps) {
		t.Fail()
	}

	commonRatio = 30 // commonFromRatio = 3
	ps = commonPeers(copyOurPeers(our), ppMap, sender, commonMax, commonRatio)
	if len(ps) != our-common+(common*commonRatio/100)-1 {
		t.Errorf("wrong peers count %d", len(ps))
	} else if false == verify(ps) {
		t.Fail()
	}

	commonRatio = 0
	ps = commonPeers(copyOurPeers(our), ppMap, sender, commonMax, commonRatio)
	if len(ps) != our-common {
		t.Errorf("wrong peers count %d", len(ps))
	} else if false == verify(ps) {
		t.Fail()
	}
}

func TestCommonPeers2(t *testing.T) {
	const totalPeers = 5
	var commonMax = 3
	var commonRatio = 10
	var ps = make([]broadcastPeer, totalPeers)
	for i := range ps {
		ps[i] = newMockPeer(vnode.RandomNodeID(), 0)
	}

	//our := ps[0]
	ourPeers := ps[1:]

	sender := ps[totalPeers-1]
	ppMap := make(map[peerId]struct{})
	for i := 0; i < totalPeers-1; i++ {
		ppMap[ps[i].ID()] = struct{}{}
	}

	commons := commonPeers(ourPeers, ppMap, sender.ID(), commonMax, commonRatio)
	if len(commons) != 1 {
		t.Errorf("wrong commons count: %d", len(commons))
	}
}
