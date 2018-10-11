package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

func mockNet() *Net {
	peers := newPeerSet()
	pool := newRequestPool()
	broadcaster := &broadcaster{
		peers: peers,
	}
	filter := &filter{
		records: make(map[types.Hash]*record),
	}
	receiver := &receiver{
		ready:       0,
		sFeed:       newSnapshotBlockFeed(),
		aFeed:       newAccountBlockFeed(),
		broadcaster: broadcaster,
		filter:      filter,
	}

	return &Net{
		Syncer: &syncer{
			state:   Syncdone,
			feed:    newSyncStateFeed(),
			peers:   peers,
			pool:    pool,
			running: 1,
		},
		Fetcher: &fetcher{
			filter:   filter,
			peers:    peers,
			receiver: receiver,
			pool:     pool,
			ready:    1,
		},
		Broadcaster: broadcaster,
		Receiver:    receiver,
		log:         log15.New("module", "net/mock_net"),
	}
}
