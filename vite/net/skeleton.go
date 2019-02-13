package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sort"
)

type blockID struct {
	height uint64
	hash   types.Hash
	prev   types.Hash
}

// use to record blocks received

type skeleton struct {
	snapshotChain []*blockID
	accountChain  map[types.Address][]*blockID
	//index         map[types.Hash]uint64
}

// @section helper to rank
type accountblocks []*ledger.AccountBlock

func (a accountblocks) Len() int {
	return len(a)
}

func (a accountblocks) Less(i, j int) bool {
	return a[i].Height < a[j].Height
}

func (a accountblocks) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a accountblocks) Sort() {
	sort.Sort(a)
}

type snapshotblocks []*ledger.SnapshotBlock

func (a snapshotblocks) Len() int {
	return len(a)
}

func (a snapshotblocks) Less(i, j int) bool {
	return a[i].Height < a[j].Height
}

func (a snapshotblocks) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a snapshotblocks) Sort() {
	sort.Sort(a)
}
