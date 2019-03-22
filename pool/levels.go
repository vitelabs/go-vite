package pool

import "fmt"

type level struct {
	index int
}

func (self *level) Index() int {
	return self.index
}

type accountLevel struct {
	level
	bs map[string]*bucket
}

func (self *accountLevel) Buckets() (result []Bucket) {
	for _, v := range self.bs {
		result = append(result, v)
	}
	return
}

func newAccountLevel(index int) *accountLevel {
	return &accountLevel{bs: make(map[string]*bucket), level: level{index: index}}
}
func (self *accountLevel) Snapshot() bool {
	return false
}

func (self *accountLevel) Add(b *Item) error {
	_, ok := self.bs[b.ownerWrapper]
	if !ok {
		self.bs[b.ownerWrapper] = newBucket(b.owner)
	}
	return self.bs[b.ownerWrapper].add(b)
}
func (self *accountLevel) print() {
	for k, v := range self.bs {
		fmt.Println("--------Bucket[" + k + "]----------")
		v.print()
	}

}

type snapshotLevel struct {
	level
	bu *bucket
}

func newSnapshotLevel(index int) *snapshotLevel {
	return &snapshotLevel{bu: newBucket(nil), level: level{index: index}}
}

func newLevel(snapshot bool, index int) Level {
	if snapshot {
		return newSnapshotLevel(index)
	} else {
		return newAccountLevel(index)
	}
}

func (self *snapshotLevel) Add(b *Item) error {
	return self.bu.add(b)
}

func (self *snapshotLevel) Snapshot() bool {
	return true
}

func (self *snapshotLevel) Buckets() (result []Bucket) {
	result = append(result, self.bu)
	return
}

type ownerLevel struct {
	owner string
	level int
}

type packages struct {
	ps []Package
}
