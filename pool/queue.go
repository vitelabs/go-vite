package pool

import (
	"fmt"
	"strconv"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/pkg/errors"
)

var (
	MAX_ERROR   = errors.New("arrived to max")
	REFER_ERROR = errors.New("refer not exist")
)

type Queue interface {
	AddItem(item *Item) error
	Levels() []Level
}

type Level interface {
	Buckets() []Bucket
}

type Bucket interface {
	Items() []*Item
}

type Item struct {
	*ledger.AccountBlock
	*ledger.SnapshotBlock
	height uint64
	prev   string
	hash   string
	owner  string
	refers []string
}

func (item *Item) Val() (*ledger.SnapshotBlock, *ledger.AccountBlock) {
	return item.SnapshotBlock, item.AccountBlock
}

func NewSnapshotItem(b *ledger.SnapshotBlock) *Item {
	i := &Item{SnapshotBlock: b}
	i.height = b.Height
	i.prev = b.PrevHash.String()
	i.hash = b.Hash.String()
	i.owner = "snapshot"
	for _, v := range b.SnapshotContent {
		i.refers = append(i.refers, v.Hash.String())
	}
	i.refers = append(i.refers, i.prev)
	return i
}

func NewAccountItem(b *ledger.AccountBlock) *Item {
	i := &Item{AccountBlock: b}
	i.height = b.Height
	i.prev = b.PrevHash.String()
	i.hash = b.Hash.String()
	i.owner = b.AccountAddress.String()
	if b.IsReceiveBlock() {
		i.refers = append(i.refers, b.FromBlockHash.String())
	}
	i.refers = append(i.refers, i.prev)
	return i
}

type bucket struct {
	bs   []*Item
	last int
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
	bu, ok := self.bs[b.owner]
	if !ok {
		self.bs[b.owner] = newBucket()
		bu = self.bs[b.owner]
	}
	return bu.add(b)
}
func (self *level) print() {
	for k, v := range self.bs {
		fmt.Println("--------Bucket[" + k + "]----------")
		v.print()
	}

}
func newBucket() *bucket {
	return &bucket{last: -1}
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

type ExistsFunc func(hash string) error

func NewQueue(f ExistsFunc) *queue {
	tmpLs := make([]*level, 10)
	for i := 0; i < 10; i++ {
		tmpLs[i] = newLevel()
	}
	return &queue{all: make(map[string]*ownerLevel), ls: tmpLs}
}

func (self *queue) add(b *Item) error {
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
			if tmp.owner == b.owner {
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
	self.all[b.hash] = &ownerLevel{b.owner, max}
	return self.ls[max].add(b)
}
func (self *queue) print() {
	for i, v := range self.ls {
		fmt.Println("----------Level" + strconv.Itoa(i) + "------------")
		v.print()
	}
}
