package batch

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

var packageId uint64

type batchSnapshot struct {
	num             int // default 0
	current         int // default -1
	lastSnapshot    int // default -1
	version         uint64
	all             map[types.Hash]*ownerLevel
	ls              []Level
	snapshotExistsF SnapshotExistsFunc
	accountExistsF  AccountExistsFunc
	maxLevel        int
	snapshot        *ledger.SnapshotBlock
	id              uint64
}

func (self *batchSnapshot) Exists(hash types.Hash) bool {
	_, result := self.all[hash]
	return result
}

func (self *batchSnapshot) Info() string {
	levelInfo := ""
	if self.Levels() == nil {
		return "level-empty"
	}
	sum := 0
	for max, l := range self.Levels() {
		levelInfo += "\n"
		buckets := l.Buckets()
		if len(buckets) == 0 {
			levelInfo = strconv.Itoa(max) + ":" + levelInfo
			break
		}
		bucketInfo := "" + strconv.Itoa(l.Index())
		for _, b := range buckets {
			bucketInfo += "|" + strconv.Itoa(len(b.Items()))
			for _, v := range b.Items() {
				bucketInfo += ":" + v.Hash().String()
				sum++
			}
		}
		levelInfo += bucketInfo
	}
	levelInfo += "\nall:"
	for k := range self.all {
		levelInfo += "|" + k.String()
	}

	return fmt.Sprintf("sum:%d,%d,%d:%s\n", len(self.all), sum, self.num, levelInfo)
}
func (self *batchSnapshot) Version() uint64 {
	return self.version
}

func (self *batchSnapshot) Size() int {
	return len(self.all)
}

func (self *batchSnapshot) IsUnconfirmed() bool {
	return self.snapshot != nil
}

type SnapshotExistsFunc func(hash types.Hash) error
type AccountExistsFunc func(hash types.Hash) error

func NewBatch(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, version uint64, max int) Batch {
	tmpLs := make([]Level, max)
	//for i := 0; i < max; i++ {
	//	tmpLs[i] = newLevel()
	//}
	id := atomic.AddUint64(&packageId, 1)
	return &batchSnapshot{all: make(map[types.Hash]*ownerLevel), ls: tmpLs, snapshotExistsF: snapshotF, accountExistsF: accountF, maxLevel: max, current: -1, lastSnapshot: -1, version: version, id: id}
}

func (self *batchSnapshot) Id() uint64 {
	return self.id
}

func (self *batchSnapshot) Levels() []Level {
	var levels []Level
	for _, v := range self.ls {
		if v == nil {
			break
		}
		levels = append(levels, v)
	}
	return levels
}

func (self *batchSnapshot) AddItem(b Item) error {
	if b.Owner() == nil {
		return self.addSnapshotItem(b)
	} else {
		return self.addAccountItem(b)
	}
}

func (self *batchSnapshot) addSnapshotItem(b Item) error {
	keys, accounts, sHash := b.ReferHashes()
	owner := b.Owner()
	// account levelInfo
	for _, r := range accounts {
		_, ok := self.all[r]
		if !ok {
			err := self.accountExistsF(r)
			if err != nil {
				return errors.WithMessage(REFER_ERROR, fmt.Sprintf("[S]account[%s] not exist.", r))
			}
		}
	}
	if sHash != nil {
		_, ok := self.all[*sHash]
		if !ok {
			err := self.snapshotExistsF(*sHash)
			if err != nil {
				return errors.WithMessage(REFER_ERROR, fmt.Sprintf("[S]snapshot[%s] not exist.", sHash))
			}
		}
	}

	max := self.current
	if max < 0 {
		max = 0
		self.ls[max] = newLevel(owner == nil, max)
	}

	tmp := self.ls[max]

	if !tmp.Snapshot() {
		max = self.current + 1
		if max > self.maxLevel-1 {
			return MAX_ERROR
		}
		tmp = newLevel(owner == nil, max)
		self.ls[max] = tmp
	}

	err := tmp.Add(b)
	if err == nil {
		self.num = self.num + 1
		self.addToAll(keys, &ownerLevel{owner, max})
	}
	if max > self.current {
		self.current = max
	}
	for i := max - 1; i > self.lastSnapshot; i-- {
		self.ls[i].Close()
	}
	self.lastSnapshot = max
	return err
}

func (self *batchSnapshot) addAccountItem(b Item) error {
	max := -1
	owner := b.Owner()
	keys, accounts, _ := b.ReferHashes()
	// account levelInfo
	for _, r := range accounts {
		tmp, ok := self.all[r]
		if !ok {
			err := self.accountExistsF(r)
			if err != nil {
				return errors.WithMessage(REFER_ERROR, fmt.Sprintf("[A]account[%s] not exist.", r))
			}
			continue
		}
		lNum := tmp.level
		lv := self.ls[lNum]

		if lv.Closed() {
			continue
		}

		if lNum >= max {
			if *tmp.owner == *owner {
				max = lNum
			} else {
				max = lNum + 1
			}
		}
		if max > self.maxLevel-1 {
			return MAX_ERROR
		}
	}
	if max < 0 {
		max = self.lastSnapshot + 1
	}
	if max > self.maxLevel-1 {
		return MAX_ERROR
	}

	var tmp Level
	if self.current >= 0 {
		tmp = self.ls[self.current]
		if tmp.Snapshot() {
			max = self.current + 1
			if max > self.maxLevel-1 {
				return MAX_ERROR
			}
			tmp = newLevel(false, max)
			self.ls[max] = tmp
		} else {
			if self.current > max {
				tmp = self.ls[max]
			}
		}
	} else {
		// first account block
		max = 0
		tmp = newLevel(false, max)
		self.ls[max] = tmp
	}

	err := tmp.Add(b)
	if err == nil {
		self.num = self.num + 1
		self.addToAll(keys, &ownerLevel{owner, max})
	}

	if max > self.current {
		self.current = max
	}
	return err
}

func (self *batchSnapshot) addToAll(keys []types.Hash, l *ownerLevel) {
	for _, v := range keys {
		self.all[v] = l
	}
}

func (self *batchSnapshot) Batch(snapshotFn BucketExecutorFn, accountFn BucketExecutorFn) error {
	executor := newBatchExecutor(self, snapshotFn, accountFn)
	return executor.execute()
}
