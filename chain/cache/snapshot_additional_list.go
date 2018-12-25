package chain_cache

import (
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

type Fragment struct {
	HeadHeight uint64
	TailHeight uint64
	List       []*AdditionItem
}

type AdditionItem struct {
	Quota              uint64
	AggregateQuota     uint64
	SnapshotHashHeight *ledger.HashHeight
}

type AdditionList struct {
	list  []*AdditionItem
	frags []*Fragment

	aggregateHeight uint64
	saveHeight      int
	flushInterval   time.Duration

	chain Chain
}

func NewAdditionList(chain Chain) (*AdditionList, error) {
	al := &AdditionList{
		aggregateHeight: 60 * 60,
		saveHeight:      5 * 24 * 60 * 60,

		flushInterval: time.Minute * 20,
		chain:         chain,
	}

	var err error
	al.list, err = al.readFromDb()

	if err != nil {
		return nil, err
	}

	return al, nil
}

func (al *AdditionList) Build() error {
	latestSnapshotBlock := al.chain.GetLatestSnapshotBlock()
	latestHeight := latestSnapshotBlock.Height

	// prepend
	prependTailHeight := uint64(1)
	if latestHeight > uint64(al.saveHeight) {
		prependTailHeight = latestHeight - uint64(al.saveHeight) + 1
	}

	prependHeadHeight := latestHeight

	if len(al.list) > 0 {
		prependHeadHeight = al.list[0].SnapshotHashHeight.Height - 1
	}

	if prependTailHeight < prependHeadHeight {
		needBuildPrependTailHeight := uint64(1)
		if prependTailHeight > al.aggregateHeight {
			needBuildPrependTailHeight = prependTailHeight - al.aggregateHeight + 1
		}
		count := prependHeadHeight - needBuildPrependTailHeight + 1
		snapshotBlocks, err := al.chain.GetSnapshotBlocksByHeight(needBuildPrependTailHeight, count, true, true)
		if err != nil {
			return err
		}

		originList := al.list
		al.list = make([]*AdditionItem, 0)
		if err := al.addList(snapshotBlocks); err != nil {
			return err
		}

		al.list = append(al.list, originList...)
	}

	// append
	appendTailHeight := uint64(0)
	appendHeadHeight := uint64(0)
	if len(al.list) > 0 {
		appendTailHeight = al.list[len(al.list)-1].SnapshotHashHeight.Height + 1
		if appendTailHeight <= latestHeight {
			appendHeadHeight = latestHeight
		}
	}

	if appendTailHeight > 0 &&
		appendHeadHeight > 0 &&
		appendHeadHeight >= appendTailHeight {

		count := appendHeadHeight - appendTailHeight + 1
		snapshotBlocks, err := al.chain.GetSnapshotBlocksByHeight(appendTailHeight, count, true, true)
		if err != nil {
			return err
		}
		al.addList(snapshotBlocks)
	}
	return nil
}

func (al *AdditionList) flush() {
	if len(al.list) <= 0 {
		return
	}

	headAdditionItem := al.list[len(al.list)-1]
	tailAdditionItem := al.list[0]

	NewFragTailHeight := tailAdditionItem.SnapshotHashHeight.Height
	if len(al.frags) > 0 {
		savedFragHeadHeight := al.frags[len(al.frags)-1].HeadHeight

		NewFragTailHeight = savedFragHeadHeight + 1
	}
	NewFragHeadHeight := headAdditionItem.SnapshotHashHeight.Height

	if NewFragTailHeight > NewFragHeadHeight {
		return
	}

	newFragListStartIndex := al.getIndexByHeight(NewFragTailHeight)
	newFragListEndIndex := al.getIndexByHeight(NewFragTailHeight)

	newFragList := al.list[newFragListStartIndex : newFragListEndIndex+1]

	newFrag := &Fragment{
		HeadHeight: NewFragHeadHeight,
		TailHeight: NewFragTailHeight,
		List:       newFragList,
	}

	al.saveFrag(newFrag)
	al.frags = append(al.frags, newFrag)
	al.clearStaleData()
}

func (al *AdditionList) clearStaleData() {
	count := len(al.list)
	if count <= 0 {
		return
	}

	if count <= al.saveHeight {
		return
	}

	needClearCount := count - al.saveHeight
	needClearAdditionItem := al.list[needClearCount-1]

	needClearFrags := make([]*Fragment, 0)
	needClearIndex := 0

	for index, frag := range al.frags {
		if frag.HeadHeight <= needClearAdditionItem.SnapshotHashHeight.Height {
			needClearFrags = append(needClearFrags, frag)
			needClearIndex = index
		}
	}

	if len(needClearFrags) <= 0 {
		return
	}

	if err := al.deleteFrags(needClearFrags); err != nil {
		// log
		return
	}
	al.frags = al.frags[needClearIndex+1:]
	al.list = al.list[needClearCount:]
}

func (al *AdditionList) deleteFrags(fragments []*Fragment) error {

	return nil
}

func (al *AdditionList) saveFrag(fragment *Fragment) error {
	return nil
}

func (al *AdditionList) readFromDb() ([]*AdditionItem, error) {
	return nil, nil
}

// TODO
func (al *AdditionList) addList(snapshotBlocks []*ledger.SnapshotBlock) error {
	for _, snapshotBlock := range snapshotBlocks {
		subLedger, err := al.chain.GetConfirmSubLedgerBySnapshotBlocks([]*ledger.SnapshotBlock{snapshotBlock})
		if err != nil {
			return err
		}

		quota := uint64(0)
		for _, blocks := range subLedger {
			for _, block := range blocks {
				quota += block.Quota
			}
		}
		al.Add(snapshotBlock, quota)
	}
}
func (al *AdditionList) Add(block *ledger.SnapshotBlock, quota uint64) {
	aggregateQuota := al.calculateQuota(block, quota)
	ai := &AdditionItem{
		Quota:          quota,
		AggregateQuota: aggregateQuota,
		SnapshotHashHeight: &ledger.HashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		},
	}
	al.list = append(al.list, ai)
}

func (al *AdditionList) calculateQuota(block *ledger.SnapshotBlock, quota uint64) uint64 {
	if block.Height <= 1 {
		return quota
	}

	prevHeight := block.Height - 1
	prevAdditionItem := al.getByHeight(prevHeight)

	if prevAdditionItem == nil {
		return quota
	}

	if block.Height <= al.aggregateHeight {

		aggregateQuota := prevAdditionItem.AggregateQuota + quota
		return aggregateQuota
	}

	tailAdditionItem := al.getByHeight(prevHeight - al.aggregateHeight + 1)
	if tailAdditionItem == nil {
		return prevAdditionItem.AggregateQuota + quota
	}

	aggregateQuota := prevAdditionItem.AggregateQuota - tailAdditionItem.Quota + quota
	return aggregateQuota
}

func (al *AdditionList) getIndexByHeight(height uint64) int {
	if len(al.list) <= 0 {
		return -1
	}
	headAdditionItem := al.list[len(al.list)-1]
	tailAdditionItem := al.list[0]

	headHeight := headAdditionItem.SnapshotHashHeight.Height
	if headHeight < height {
		return -1
	}

	tailHeight := tailAdditionItem.SnapshotHashHeight.Height
	if tailHeight > height {
		return -1
	}

	index := int(height - tailHeight)
	return index

}
func (al *AdditionList) getByHeight(height uint64) *AdditionItem {
	index := al.getIndexByHeight(height)
	if index < 0 {
		return nil
	}
	return al.list[index]
}
