package pool

import (
	"fmt"
	"strconv"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
)

type snapshotPackage struct {
	all             map[types.Hash]*ownerLevel
	ls              []*level
	snapshotExistsF SnapshotExistsFunc
	accountExistsF  AccountExistsFunc
	maxLevel        int
	snapshot        *ledger.SnapshotBlock
}

func (self *snapshotPackage) Info() string {
	levelInfo := ""
	for max, l := range self.Levels() {
		levelInfo += "\n"
		buckets := l.Buckets()
		if len(buckets) == 0 {
			levelInfo = strconv.Itoa(max) + ":" + levelInfo
			break
		}
		bucketInfo := ""
		for _, b := range buckets {
			bucketInfo += "|" + strconv.Itoa(len(b.Items()))
		}
		levelInfo += bucketInfo
	}

	return fmt.Sprintf("sum:%d, level:%s\n", len(self.all), levelInfo)
}

func (self *snapshotPackage) Size() int {
	return len(self.all)
}

func (self *snapshotPackage) IsUnconfirmed() bool {
	return self.snapshot != nil
}

type SnapshotExistsFunc func(hash types.Hash) error
type AccountExistsFunc func(hash types.Hash) error

func NewSnapshotPackage(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, max int) Package {
	tmpLs := make([]*level, max)
	for i := 0; i < max; i++ {
		tmpLs[i] = newLevel()
	}
	return &snapshotPackage{all: make(map[types.Hash]*ownerLevel), ls: tmpLs, snapshotExistsF: snapshotF, accountExistsF: accountF, maxLevel: max}
}
func NewSnapshotPackage2(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, max int, snapshot *ledger.SnapshotBlock) *snapshotPackage {
	tmpLs := make([]*level, max)
	for i := 0; i < max; i++ {
		tmpLs[i] = newLevel()
	}
	return &snapshotPackage{all: make(map[types.Hash]*ownerLevel), ls: tmpLs, snapshotExistsF: snapshotF, accountExistsF: accountF, maxLevel: max, snapshot: snapshot}
}

func (self *snapshotPackage) Levels() []Level {
	var levels []Level
	for _, v := range self.ls {
		levels = append(levels, v)
	}
	return levels
}

func (self *snapshotPackage) AddItem(b *Item) error {
	max := 0

	// account level
	for _, r := range b.referAccounts {
		tmp, ok := self.all[r]
		if !ok {
			err := self.accountExistsF(r)
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
		if max > self.maxLevel-1 {
			return MAX_ERROR
		}
	}
	// snapshot level
	if b.referSnapshot != nil {
		sHash := *b.referSnapshot
		tmp, ok := self.all[sHash]
		if !ok {
			err := self.snapshotExistsF(sHash)
			if err != nil {
				return REFER_ERROR
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
	self.all[b.Hash()] = &ownerLevel{b.ownerWrapper, max}
	return self.ls[max].add(b)
}
func (self *snapshotPackage) print() {
	for i, v := range self.ls {
		fmt.Println("----------Level" + strconv.Itoa(i) + "------------")
		v.print()
	}
}
