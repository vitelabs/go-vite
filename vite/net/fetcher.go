package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync/atomic"
)

type Fetcher interface {
	FetchSnapshotBlocks(start types.Hash, count uint64)
	// address may be nil
	FetchAccountBlocks(start types.Hash, count uint64, address *types.Address)
}

type fetcher struct {
	filter   Filter
	peers    *peerSet
	receiver Receiver
	pool     RequestPool
	ready    int32 // atomic
}

func newFetcher(filter Filter, peers *peerSet, receiver Receiver, pool RequestPool) *fetcher {
	return &fetcher{
		filter:   filter,
		peers:    peers,
		receiver: receiver,
		pool:     pool,
	}
}

func (f *fetcher) FetchSnapshotBlocks(start types.Hash, count uint64) {
	panic("implement me")
}

func (f *fetcher) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	panic("implement me")
}

func (f *fetcher) listen(st SyncState) {
	if st == Syncdone || st == SyncDownloaded {
		atomic.StoreInt32(&f.ready, 1)

	}
}
