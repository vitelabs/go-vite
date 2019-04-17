package chain_cache

import (
	"container/list"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type quotaInfo struct {
	BlockCount uint64
	Quota      uint64
}
type quotaList struct {
	chain Chain

	backElement map[types.Address]*quotaInfo

	used                 map[types.Address]*quotaInfo
	usedStart            *list.Element
	usedAccumulateHeight uint64

	list          *list.List
	listMaxLength int

	status byte
}

func newQuotaList(chain Chain) *quotaList {
	ql := &quotaList{
		chain: chain,
		used:  make(map[types.Address]*quotaInfo),

		backElement: make(map[types.Address]*quotaInfo),

		list:                 list.New(),
		listMaxLength:        600,
		usedAccumulateHeight: 75,
	}

	return ql
}

func (ql *quotaList) init() error {
	if err := ql.build(); err != nil {
		return err
	}

	ql.status = 1
	return nil
}

func (ql *quotaList) GetSnapshotQuotaUsed(addr *types.Address) (uint64, uint64) {
	used := ql.used[*addr]
	if used == nil {
		return 0, 0
	}
	quota := used.Quota
	blockCount := used.BlockCount
	latestUsed := ql.backElement[*addr]
	if latestUsed != nil {
		return quota - latestUsed.Quota, blockCount - latestUsed.BlockCount
	}

	return quota, blockCount
}

func (ql *quotaList) GetQuotaUsed(addr *types.Address) (uint64, uint64) {
	used := ql.used[*addr]
	if used == nil {
		return 0, 0
	}
	return used.Quota, used.BlockCount
}

func (ql *quotaList) Add(addr types.Address, quota uint64) {
	// add back element quota
	ql.add(ql.backElement, addr, quota)

	// add used quota
	ql.add(ql.used, addr, quota)

}

func (ql *quotaList) Sub(addr types.Address, quota uint64) {

	// add back element
	ql.sub(ql.backElement, addr, 1, quota)

	// sub used quota
	ql.sub(ql.used, addr, 1, quota)

}

func (ql *quotaList) NewEmptyNext() {
	if ql.status < 1 {
		return
	}

	ql.list.PushBack(make(map[types.Address]*quotaInfo))

	ql.backElement = ql.list.Back().Value.(map[types.Address]*quotaInfo)

	ql.moveNext()
}

func (ql *quotaList) NewNext(confirmedBlocks []*ledger.AccountBlock) {
	if ql.status < 1 {
		return
	}

	currentSnapshotQuota := make(map[types.Address]*quotaInfo)

	for _, confirmedBlock := range confirmedBlocks {
		qi, ok := currentSnapshotQuota[confirmedBlock.AccountAddress]
		backQi := ql.backElement[confirmedBlock.AccountAddress]

		if !ok {
			qi = &quotaInfo{}
			currentSnapshotQuota[confirmedBlock.AccountAddress] = qi
		}
		qi.Quota += confirmedBlock.Quota
		qi.BlockCount += 1

		if backQi.BlockCount <= 1 {
			delete(ql.backElement, confirmedBlock.AccountAddress)
		} else {
			backQi.BlockCount -= 1
			backQi.Quota -= confirmedBlock.Quota
		}
	}

	ql.list.Back().Value = currentSnapshotQuota
	ql.list.PushBack(ql.backElement)
	ql.moveNext()

}

func (ql *quotaList) moveNext() {
	if uint64(ql.list.Len()) <= ql.usedAccumulateHeight {
		return
	}

	quotaUsedStart := ql.usedStart.Value.(map[types.Address]*quotaInfo)
	for addr, usedStartItem := range quotaUsedStart {
		if usedStartItem == nil {
			continue
		}
		ql.sub(ql.used, addr, usedStartItem.BlockCount, usedStartItem.Quota)
	}

	ql.usedStart = ql.usedStart.Next()
	if ql.list.Len() > ql.listMaxLength {
		ql.list.Remove(ql.list.Front())
	}

}

func (ql *quotaList) Rollback(deletedChunks []*ledger.SnapshotChunk, hasStorageRedoLog bool) error {
	backElem := ql.list.Back()
	if backElem == nil {
		return nil
	}
	if len(backElem.Value.(map[types.Address]*quotaInfo)) <= 0 {
		ql.list.Remove(backElem)
	}

	n := len(deletedChunks)

	if n >= ql.listMaxLength {
		ql.list.Init()
	} else {
		if hasStorageRedoLog {
			n = n - 1
		}

		for i := 0; i < n && ql.list.Len() > 0; i++ {
			ql.list.Remove(ql.list.Back())
		}
	}

	ql.build()

	return nil
}

func (ql *quotaList) build() (returnError error) {
	defer func() {
		if returnError != nil {
			return
		}
		ql.backElement = ql.list.Back().Value.(map[types.Address]*quotaInfo)

		ql.resetUsedStart()

		ql.calculateUsed()

		//fmt.Println("after build, ql.backElement", ql.list.Len())
		//for addr, quotaInfo := range ql.backElement {
		//	fmt.Println(addr, quotaInfo)
		//}
		//fmt.Println()
	}()

	listLength := uint64(ql.list.Len())

	// FOR DEBUG
	//fmt.Println("before build, ql.backElement", listLength)
	//for addr, quotaInfo := range ql.backElement {
	//	fmt.Println(addr, quotaInfo)
	//}
	//fmt.Println()

	if listLength >= ql.usedAccumulateHeight {
		return nil
	}

	latestSb, err := ql.chain.QueryLatestSnapshotBlock()
	if err != nil {
		return err
	}

	latestSbHeight := latestSb.Height

	if latestSbHeight <= listLength {
		return nil
	}

	endSbHeight := latestSbHeight + 1 - listLength
	startSbHeight := uint64(1)

	lackListLen := uint64(ql.listMaxLength) - listLength
	if endSbHeight > lackListLen {
		startSbHeight = endSbHeight - lackListLen
	}

	var snapshotSegments []*ledger.SnapshotChunk

	if listLength <= 0 {
		snapshotSegments, err = ql.chain.GetSubLedgerAfterHeight(startSbHeight)
		if err != nil {
			return err
		}

		if snapshotSegments == nil {
			return errors.New(fmt.Sprintf("ql.chain.GetSubLedgerAfterHeight, snapshotSegments is nil, startSbHeight is %d", startSbHeight))
		}

		for _, seg := range snapshotSegments[1:] {

			newItem := make(map[types.Address]*quotaInfo)
			for _, block := range seg.AccountBlocks {
				if _, ok := newItem[block.AccountAddress]; !ok {
					newItem[block.AccountAddress] = &quotaInfo{
						Quota:      block.Quota,
						BlockCount: 1,
					}
				} else {
					newItem[block.AccountAddress].Quota += block.Quota
					newItem[block.AccountAddress].BlockCount += 1
				}

			}
			ql.list.PushBack(newItem)
		}

		if snapshotSegments[len(snapshotSegments)-1].SnapshotBlock != nil {
			ql.list.PushBack(make(map[types.Address]*quotaInfo))
		}

	} else {
		snapshotSegments, err = ql.chain.GetSubLedger(startSbHeight, endSbHeight)
		if err != nil {
			return err
		}

		if snapshotSegments == nil {
			return errors.New(fmt.Sprintf("ql.chain.GetSubLedger, snapshotSegments is nil, startSbHeight is %d, endSbHeight is %d",
				startSbHeight, endSbHeight))
		}

		segLength := len(snapshotSegments)
		for i := segLength - 1; i > 0; i-- {
			seg := snapshotSegments[i]
			newItem := make(map[types.Address]*quotaInfo)

			for _, block := range seg.AccountBlocks {
				if _, ok := newItem[block.AccountAddress]; !ok {
					newItem[block.AccountAddress] = &quotaInfo{
						Quota:      block.Quota,
						BlockCount: 1,
					}
				} else {
					newItem[block.AccountAddress].Quota += block.Quota
					newItem[block.AccountAddress].BlockCount += 1
				}
			}
			ql.list.PushFront(newItem)
		}

	}

	return nil
}

func (ql *quotaList) add(quotaInfoMap map[types.Address]*quotaInfo, addr types.Address, quota uint64) {
	qi := quotaInfoMap[addr]
	if qi == nil {
		qi = &quotaInfo{}
		quotaInfoMap[addr] = qi
	}
	qi.BlockCount += 1
	qi.Quota += quota
}

func (ql *quotaList) sub(quotaInfoMap map[types.Address]*quotaInfo, addr types.Address, blockCount, quota uint64) {
	qi := quotaInfoMap[addr]
	if qi == nil {
		return
	}
	if qi.BlockCount <= blockCount {
		delete(quotaInfoMap, addr)
	} else {
		qi.BlockCount -= blockCount
		qi.Quota -= quota
		return
	}

}

func (ql *quotaList) calculateUsed() {
	used := make(map[types.Address]*quotaInfo)

	pointer := ql.usedStart
	for pointer != nil {
		tmpUsed := pointer.Value.(map[types.Address]*quotaInfo)
		for addr, tmpItem := range tmpUsed {
			if used[addr] == nil {
				used[addr] = &quotaInfo{}
			}

			used[addr].BlockCount += tmpItem.BlockCount
			used[addr].Quota += tmpItem.Quota
		}

		pointer = pointer.Next()
	}
	ql.used = used
}

func (ql *quotaList) resetUsedStart() {
	ql.usedStart = ql.list.Back()

	for i := uint64(1); i < ql.usedAccumulateHeight; i++ {
		prev := ql.usedStart.Prev()
		if prev == nil {
			break
		}
		ql.usedStart = prev

	}
}
