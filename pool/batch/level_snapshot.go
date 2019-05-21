package batch

import "github.com/go-errors/errors"

type snapshotLevel struct {
	level
	bu *bucket
}

func newSnapshotLevel(index int) *snapshotLevel {
	return &snapshotLevel{bu: newBucket(nil), level: level{index: index}}
}

func (self *snapshotLevel) Add(b Item) error {
	if self.Closed() {
		panic(errors.New("snapshot level is closed."))
	}
	return self.bu.add(b)
}

func (self *snapshotLevel) Snapshot() bool {
	return true
}

func (self *snapshotLevel) Buckets() (result []Bucket) {
	result = append(result, self.bu)
	return
}
