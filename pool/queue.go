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

type Queue interface {
	AddItem(item *Item) error
	Levels() []Level
	Size() int
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
	height       uint64
	prev         string
	hash         string
	owner        *types.Address
	ownerWrapper string
	refers       []string
}

func NewItem(b commonBlock, owner *types.Address) *Item {
	i := &Item{commonBlock: b}
	i.height = b.Height()
	i.prev = b.PrevHash().String()
	i.hash = b.Hash().String()
	i.owner = owner
	if owner == nil {
		i.ownerWrapper = "snapshot"
	} else {
		i.ownerWrapper = owner.String()
	}
	for _, v := range b.ReferHashes() {
		i.refers = append(i.refers, v.String())
	}
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
		if lastB.hash != b.prev {
			return errors.New("prev and hash")
		}
	}

	self.bs = append(self.bs, b)
	self.last = self.last + 1
	return nil
}
func (self *bucket) print() {
	for _, v := range self.bs {
		fmt.Print(strconv.FormatUint(v.height, 10) + ",")
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

type queue struct {
	all     map[string]*ownerLevel
	ls      []*level
	existsF ExistsFunc
}

func (self *queue) Size() int {
	return len(self.all)
}

type ExistsFunc func(hash string) error

func NewQueue(f ExistsFunc) Queue {
	tmpLs := make([]*level, 50)
	for i := 0; i < 50; i++ {
		tmpLs[i] = newLevel()
	}
	return &queue{all: make(map[string]*ownerLevel), ls: tmpLs, existsF: f}
}

func (self *queue) Levels() []Level {
	var levels []Level
	for _, v := range self.ls {
		levels = append(levels, v)
	}
	return levels
}

func (self *queue) AddItem(b *Item) error {
	max := 0
	for _, r := range b.refers {
		tmp, ok := self.all[r]
		if !ok {
			err := self.existsF(r)
			if err != nil {
				return REFER_ERROR
			}
			continue
		}
		lNum := tmp.level
		if lNum >= max {
			if tmp.owner == b.ownerWrapper {
				max = lNum
			} else {
				max = lNum + 1
			}
		}
		if max > 9 {
			return MAX_ERROR
		}
	}
	if max > 9 {
		return MAX_ERROR
	}
	self.all[b.hash] = &ownerLevel{b.ownerWrapper, max}
	return self.ls[max].add(b)
}
func (self *queue) print() {
	for i, v := range self.ls {
		fmt.Println("----------Level" + strconv.Itoa(i) + "------------")
		v.print()
	}
}
