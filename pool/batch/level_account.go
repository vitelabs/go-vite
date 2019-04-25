package batch

import (
	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
)

type accountLevel struct {
	level
	bs map[types.Address]*bucket
}

func (self *accountLevel) Buckets() (result []Bucket) {
	for _, v := range self.bs {
		result = append(result, v)
	}
	return
}

func newAccountLevel(index int) *accountLevel {
	return &accountLevel{bs: make(map[types.Address]*bucket), level: level{index: index}}
}

func (self *accountLevel) Snapshot() bool {
	return false
}

func (self *accountLevel) Add(b Item) error {
	if self.Closed() {
		panic(errors.New("level is closed"))
	}
	owner := *b.Owner()
	_, ok := self.bs[owner]
	if !ok {
		self.bs[owner] = newBucket(&owner)
	}
	return self.bs[owner].add(b)
}
