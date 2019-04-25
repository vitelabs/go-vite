package batch

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/vitelabs/go-vite/common/types"
)

type bucket struct {
	bs    []Item
	last  int
	owner *types.Address
}

func (self *bucket) Owner() *types.Address {
	return self.owner
}

func (self *bucket) Items() []Item {
	return self.bs
}

func (self *bucket) add(b Item) error {
	if self.last == -1 && len(self.bs) != 0 {
		return errors.New("bucket must be empty")
	}

	if self.last != -1 {
		lastB := self.bs[self.last]
		if lastB == nil {
			return errors.New("lastB must be exists")
		}
		if lastB.Hash() != b.PrevHash() {
			return errors.New("prev and hash")
		}
	}

	self.bs = append(self.bs, b)
	self.last = self.last + 1
	return nil
}
func (self *bucket) print() {
	for _, v := range self.bs {
		fmt.Print(strconv.FormatUint(v.Height(), 10) + ",")
	}
	fmt.Println()
}

func newBucket(owner *types.Address) *bucket {
	return &bucket{last: -1, owner: owner}
}
