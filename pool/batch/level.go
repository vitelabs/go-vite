package batch

import "github.com/vitelabs/go-vite/common/types"

type level struct {
	index  int
	closed bool
}

func (self *level) Index() int {
	return self.index
}

func (self *level) Close() {
	if self.closed {
		panic("has closed")
	}
	self.closed = true
}

func (self *level) Closed() bool {
	return self.closed
}

func newLevel(snapshot bool, index int) Level {
	if snapshot {
		return newSnapshotLevel(index)
	} else {
		return newAccountLevel(index)
	}
}

type ownerLevel struct {
	owner *types.Address
	level int
}
