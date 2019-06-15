package pool

import (
	"strconv"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite/net"
)

type tools struct {
	// if address == nil, snapshot tools
	// else account fetcher
	fetcher commonSyncer
	rw      chainRw
}

func newTools(f commonSyncer, rw chainRw) *tools {
	self := &tools{}
	self.fetcher = f
	self.rw = rw
	return self
}

type syncer interface {
	net.Broadcaster
	net.Fetcher
	net.Subscriber
	net.ChunkReader
}

type fetchRequest struct {
	snapshot       bool
	chain          *types.Address
	hash           types.Hash
	accHeight      uint64
	prevCnt        uint64
	snapshotHash   *types.Hash
	snapshotHeight uint64
}

func (request *fetchRequest) String() string {
	if request.chain == nil {
		return strconv.FormatBool(request.snapshot) + "," + request.hash.String() + "," + strconv.FormatUint(request.prevCnt, 10)
	}
	return strconv.FormatBool(request.snapshot) + "," + request.hash.String() + "," + strconv.FormatUint(request.prevCnt, 10) + request.chain.String() + ","
}
