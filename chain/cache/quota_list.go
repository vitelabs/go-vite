package chain_cache

import (
	"container/list"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type quotaInfo types.QuotaInfo

type quotaList struct {
	chain Chain

	backElement map[types.Address]*quotaInfo

	globalUsed types.QuotaInfo

	usedStart *list.Element

	usedAccumulateHeight int

	list          *list.List
	listMaxLength int

	status byte
	log    log15.Logger
}

func newQuotaList(chain Chain) *quotaList {
	ql := &quotaList{
		chain: chain,

		backElement: make(map[types.Address]*quotaInfo),

		list:                 list.New(),
		listMaxLength:        600,
		usedAccumulateHeight: 75,

		log: log15.New("module", "quota_list"),
	}

	return ql
}

func (ql *quotaList) init() error {
	if err := ql.build(); err != nil {
		return err
	}
	ql.moveNext(make(map[types.Address]*quotaInfo))

	ql.status = 1
	return nil
}

func (ql *quotaList) GetGlobalQuota() types.QuotaInfo {
	globalQuota := ql.globalUsed

	for _, quotaInfo := range ql.backElement {
		globalQuota.BlockCount -= quotaInfo.BlockCount
		globalQuota.QuotaTotal -= quotaInfo.QuotaTotal
		globalQuota.QuotaUsedTotal -= quotaInfo.QuotaUsedTotal
	}
	return globalQuota
}

func (ql *quotaList) GetQuotaUsedList(addr types.Address) []types.QuotaInfo {
	usedList := make([]types.QuotaInfo, 0, ql.usedAccumulateHeight)

	pointer := ql.usedStart

	for pointer != nil {
		tmpUsed := pointer.Value.(map[types.Address]*quotaInfo)

		addrUsed, ok := tmpUsed[addr]
		if ok {
			usedList = append(usedList, types.QuotaInfo(*addrUsed))
		} else {
			usedList = append(usedList, types.QuotaInfo{})
		}

		pointer = pointer.Next()
	}

	if len(usedList) <= 0 {
		ql.log.Warn(fmt.Sprintf("GetQuotaUsedList: %s, return %d list,", addr, len(usedList)), "method", "GetQuotaUsedList")
	}

	return usedList
}

func (ql *quotaList) Add(addr types.Address, quota uint64, quotaUsed uint64) {
	// add back element quota
	ql.add(ql.backElement, addr, quota, quotaUsed)

	// add globalUsed quota
	ql.globalUsed.BlockCount += 1
	ql.globalUsed.QuotaTotal += quota
	ql.globalUsed.QuotaUsedTotal += quotaUsed

}

func (ql *quotaList) Sub(addr types.Address, quota uint64, quotaUsed uint64) {

	// sub back element quota
	ql.sub(ql.backElement, addr, 1, quota, quotaUsed)

	// sub globalUsed quota
	ql.globalUsed.BlockCount -= 1
	ql.globalUsed.QuotaTotal -= quota
	ql.globalUsed.QuotaUsedTotal -= quotaUsed

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
		qi.BlockCount += 1
		qi.QuotaTotal += confirmedBlock.Quota
		qi.QuotaUsedTotal += confirmedBlock.QuotaUsed

		if backQi.BlockCount <= 1 {
			delete(ql.backElement, confirmedBlock.AccountAddress)
		} else {
			backQi.BlockCount -= 1
			backQi.QuotaTotal -= confirmedBlock.Quota
			backQi.QuotaUsedTotal -= confirmedBlock.QuotaUsed
		}
	}

	ql.list.Back().Value = currentSnapshotQuota
	ql.moveNext(ql.backElement)

}

func (ql *quotaList) Rollback(deletedChunks []*ledger.SnapshotChunk) error {
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
		for i := 0; i < n && ql.list.Len() > 0; i++ {
			ql.list.Remove(ql.list.Back())
		}
	}

	if err := ql.build(); err != nil {
		return err
	}

	ql.moveNext(make(map[types.Address]*quotaInfo))
	return nil
}

func (ql *quotaList) build() (returnError error) {
	defer func() {
		if returnError != nil {
			return
		}
		if ql.list.Len() <= 0 {
			return
		}
		ql.backElement = ql.list.Back().Value.(map[types.Address]*quotaInfo)

		ql.resetUsedStart()

		ql.calculateGlobalUsed()

	}()

	listLength := ql.list.Len()

	if listLength >= ql.usedAccumulateHeight {
		return nil
	}

	latestSb, err := ql.chain.QueryLatestSnapshotBlock()
	if err != nil {
		return err
	}

	latestSbHeight := latestSb.Height

	if latestSbHeight <= uint64(listLength) {
		return nil
	}

	endSbHeight := latestSbHeight + 1 - uint64(listLength)
	startSbHeight := uint64(1)

	lackListLen := uint64(ql.listMaxLength - listLength)
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
						QuotaTotal:     block.Quota,
						QuotaUsedTotal: block.QuotaUsed,
						BlockCount:     1,
					}
				} else {
					quotaInfo := newItem[block.AccountAddress]
					quotaInfo.QuotaTotal += block.Quota
					quotaInfo.QuotaUsedTotal += block.QuotaUsed
					quotaInfo.BlockCount += 1
				}

			}
			ql.list.PushBack(newItem)
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
						QuotaTotal:     block.Quota,
						QuotaUsedTotal: block.QuotaUsed,
						BlockCount:     1,
					}
				} else {
					quotaInfo := newItem[block.AccountAddress]
					quotaInfo.QuotaTotal += block.Quota
					quotaInfo.QuotaUsedTotal += block.QuotaUsed
					quotaInfo.BlockCount += 1
				}
			}
			ql.list.PushFront(newItem)
		}

	}

	return nil
}

func (ql *quotaList) moveNext(backElement map[types.Address]*quotaInfo) {

	ql.list.PushBack(backElement)

	ql.backElement = backElement

	if ql.usedStart == nil {
		ql.usedStart = ql.list.Back()

	}

	if ql.list.Len() <= ql.usedAccumulateHeight {
		return
	}

	quotaUsedStart := ql.usedStart.Value.(map[types.Address]*quotaInfo)

	for _, usedStartItem := range quotaUsedStart {
		if usedStartItem == nil {
			continue
		}
		ql.globalUsed.QuotaUsedTotal -= usedStartItem.QuotaUsedTotal
		ql.globalUsed.BlockCount -= usedStartItem.BlockCount
		ql.globalUsed.QuotaTotal -= usedStartItem.QuotaTotal
	}

	ql.usedStart = ql.usedStart.Next()
	if ql.list.Len() > ql.listMaxLength {
		ql.list.Remove(ql.list.Front())
	}

}

func (ql *quotaList) add(quotaInfoMap map[types.Address]*quotaInfo, addr types.Address, quota uint64, quotaUsed uint64) {
	qi := quotaInfoMap[addr]
	if qi == nil {
		qi = &quotaInfo{}
		quotaInfoMap[addr] = qi
	}
	qi.BlockCount += 1
	qi.QuotaTotal += quota
	qi.QuotaUsedTotal += quotaUsed
}

func (ql *quotaList) sub(quotaInfoMap map[types.Address]*quotaInfo, addr types.Address, blockCount, quota uint64, quotaUsed uint64) {
	qi := quotaInfoMap[addr]
	if qi == nil {
		return
	}
	if qi.BlockCount <= blockCount {
		delete(quotaInfoMap, addr)
	} else {
		qi.BlockCount -= blockCount
		qi.QuotaTotal -= quota
		qi.QuotaUsedTotal -= quotaUsed
		return
	}

}

func (ql *quotaList) calculateGlobalUsed() {
	var globalUsed types.QuotaInfo

	pointer := ql.usedStart
	for pointer != nil {
		tmpUsed := pointer.Value.(map[types.Address]*quotaInfo)
		for _, tmpItem := range tmpUsed {
			globalUsed.BlockCount += tmpItem.BlockCount
			globalUsed.QuotaTotal += tmpItem.QuotaTotal
			globalUsed.QuotaUsedTotal += tmpItem.QuotaUsedTotal
		}

		pointer = pointer.Next()
	}

	ql.globalUsed = globalUsed
}

func (ql *quotaList) resetUsedStart() {
	ql.usedStart = ql.list.Back()

	for i := 1; i < ql.usedAccumulateHeight; i++ {
		prev := ql.usedStart.Prev()
		if prev == nil {
			break
		}
		ql.usedStart = prev

	}
}
