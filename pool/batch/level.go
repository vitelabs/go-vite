package batch

import "github.com/vitelabs/go-vite/common/types"

type level struct {
	index  int
	closed bool
	done   bool
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

func (self *level) Done() {
	self.done = true
}

func (self *level) HasDone() bool {
	return self.done
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
