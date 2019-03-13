package pool

import (
	"fmt"
	"strconv"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/pkg/errors"
)

var (
	MAX_ERROR   = errors.New("arrived to max")
	REFER_ERROR = errors.New("refer not exist")
)

type Packages interface {
}

type Package interface {
	AddItem(item *Item) error
	Levels() []Level
	Size() int
	Info() string
	Exists(hash types.Hash) bool
}

type Level interface {
	Buckets() []Bucket
}

type Bucket interface {
	Items() []*Item
	Owner() *types.Address
}

type Item struct {
	commonBlock
	owner         *types.Address
	ownerWrapper  string
	referAccounts []types.Hash
	referSnapshot *types.Hash
}

func NewItem(b commonBlock, owner *types.Address) *Item {
	i := &Item{commonBlock: b}
	i.owner = owner
	if owner == nil {
		i.ownerWrapper = "snapshot"
	} else {
		i.ownerWrapper = owner.String()
	}
	i.referAccounts, i.referSnapshot = b.ReferHashes()
	return i
}

type bucket struct {
	bs    []*Item
	last  int
	owner *types.Address
}

func (self *bucket) Owner() *types.Address {
	return self.owner
}

func (self *bucket) Items() []*Item {
	return self.bs
}

func (self *bucket) add(b *Item) error {
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

type level struct {
	bs map[string]*bucket
}

func (self *level) Buckets() (result []Bucket) {
	for _, v := range self.bs {
		result = append(result, v)
	}
	return
}

func newLevel() *level {
	return &level{bs: make(map[string]*bucket)}
}

func (self *level) add(b *Item) error {
	bu, ok := self.bs[b.ownerWrapper]
	if !ok {
		self.bs[b.ownerWrapper] = newBucket(b.owner)
		bu = self.bs[b.ownerWrapper]
	}
	return bu.add(b)
}
func (self *level) print() {
	for k, v := range self.bs {
		fmt.Println("--------Bucket[" + k + "]----------")
		v.print()
	}

}

type ownerLevel struct {
	owner string
	level int
}

type packages struct {
	ps []Package
}
