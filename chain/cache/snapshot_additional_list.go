package chain_cache

import "github.com/vitelabs/go-vite/ledger"

type AdditionItem struct {
	Quota              uint64
	AggregateQuota     uint64
	SnapshotHashHeight *ledger.HashHeight
}

type AdditionList struct {
	list            []*AdditionItem
	aggregateHeight uint64
}

// TODO
func (al *AdditionList) Add(block *ledger.SnapshotBlock, quota uint64) {
	ai := &AdditionItem{
		Quota: quota,
		SnapshotHashHeight: &ledger.HashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		},
	}
	al.list = append(al.list, ai)
}

// TODO
func (al *AdditionList) getByHeight(height uint64) *AdditionItem {
	headAdditionItem := al.list[len(al.list)-1]
	tailAdditionItem := al.list[0]

	headHeight := headAdditionItem.SnapshotHashHeight.Height
	if headHeight < height {
		return nil
	}

	tailHeight := tailAdditionItem.SnapshotHashHeight.Height
	if tailHeight > height {
		return nil
	}

}
