package pool

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

var packageId uint64

type snapshotPackage struct {
	num             int
	current         int
	version         int
	all             map[types.Hash]*ownerLevel
	ls              []Level
	snapshotExistsF SnapshotExistsFunc
	accountExistsF  AccountExistsFunc
	maxLevel        int
	snapshot        *ledger.SnapshotBlock
	id              uint64
}

func (self *snapshotPackage) Exists(hash types.Hash) bool {
	_, result := self.all[hash]
	return result
}

func (self *snapshotPackage) Info() string {
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
func (self *snapshotPackage) Version() int {
	return self.version
}

func (self *snapshotPackage) Size() int {
	return len(self.all)
}

func (self *snapshotPackage) IsUnconfirmed() bool {
	return self.snapshot != nil
}

type SnapshotExistsFunc func(hash types.Hash) error
type AccountExistsFunc func(hash types.Hash) error

func NewSnapshotPackage(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, version int, max int) Package {
	tmpLs := make([]Level, max)
	//for i := 0; i < max; i++ {
	//	tmpLs[i] = newLevel()
	//}
	id := atomic.AddUint64(&packageId, 1)
	return &snapshotPackage{all: make(map[types.Hash]*ownerLevel), ls: tmpLs, snapshotExistsF: snapshotF, accountExistsF: accountF, maxLevel: max, current: -1, version: version, id: id}
}
func NewSnapshotPackage2(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, max int, snapshot *ledger.SnapshotBlock) *snapshotPackage {
	tmpLs := make([]Level, max)
	//for i := 0; i < max; i++ {
	//	tmpLs[i] = newLevel()
	//}
	return &snapshotPackage{all: make(map[types.Hash]*ownerLevel), ls: tmpLs, snapshotExistsF: snapshotF, accountExistsF: accountF, maxLevel: max, snapshot: snapshot, current: -1}
}

func (self *snapshotPackage) Id() uint64 {
	return self.id
}

func (self *snapshotPackage) Levels() []Level {
	var levels []Level
	for _, v := range self.ls {
		if v == nil {
			break
		}
		levels = append(levels, v)
	}
	return levels
}

func (self *snapshotPackage) AddItem(b *Item) error {
	max := 0

	// account levelInfo
	for _, r := range b.referAccounts {
		tmp, ok := self.all[r]
		if !ok {
			err := self.accountExistsF(r)
			if err != nil {
				return errors.WithMessage(REFER_ERROR, fmt.Sprintf("account[%s] not exist.", r))
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
		if max > self.maxLevel-1 {
			return MAX_ERROR
		}
	}
	// snapshot levelInfo
	if b.referSnapshot != nil {
		sHash := *b.referSnapshot
		tmp, ok := self.all[sHash]
		if !ok {
			err := self.snapshotExistsF(sHash)
			if err != nil {
				return errors.WithMessage(REFER_ERROR, fmt.Sprintf("snapshot[%s] not exist.", sHash))
			}
		} else {
			lNum := tmp.level
			if lNum >= max {
				if tmp.owner == b.ownerWrapper {
					max = lNum
				} else {
					max = lNum + 1
				}
			}
		}
	}
	if max > self.maxLevel-1 {
		return MAX_ERROR
	}
	tmp := self.ls[max]
	if tmp == nil {
		tmp = newLevel(b.Snapshot(), max)
		self.ls[max] = tmp
	} else {
		if b.Snapshot() {
			if !tmp.Snapshot() {
				max = self.current + 1
				if max > self.maxLevel-1 {
					return MAX_ERROR
				}
				tmp = newLevel(b.Snapshot(), max)
				self.ls[max] = tmp
			}
		} else {
			if self.current > max {
				max = self.current
				tmp = self.ls[max]
			}
			if tmp.Snapshot() {
				max = self.current + 1
				if max > self.maxLevel-1 {
					return MAX_ERROR
				}
				tmp = newLevel(b.Snapshot(), max)
				self.ls[max] = tmp
			}
		}
	}

	err := tmp.Add(b)
	if err == nil {
		self.num = self.num + 1
		self.addToAll(b, &ownerLevel{b.ownerWrapper, max})
	}
	if max > self.current {
		self.current = max
	}
	return err
}

func (self *snapshotPackage) print() {
	for i, _ := range self.ls {
		fmt.Println("----------Level" + strconv.Itoa(i) + "------------")
		//v.print()
	}
}
func (self *snapshotPackage) addToAll(b *Item, l *ownerLevel) {
	//fmt.Printf("add to item:[%s-%s-%d]%s\n", b.ownerWrapper, b.commonBlock.Hash(), b.commonBlock.Height(), b.commonBlock.Latency())
	for _, v := range b.Keys() {
		self.all[v] = l
	}
}
