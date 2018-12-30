package chain_cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type NeedSnapshotBlock struct {
	Block           *ledger.AccountBlock
	AccumulateQuota uint64
}

type NeedSnapshotCache struct {
	subLedger map[types.Address][]*NeedSnapshotBlock
	chain     Chain

	log  log15.Logger
	lock sync.Mutex
}

func NewNeedSnapshotCache(chain Chain) *NeedSnapshotCache {
	return &NeedSnapshotCache{
		subLedger: make(map[types.Address][]*NeedSnapshotBlock),
		chain:     chain,

		log: log15.New("module", "chain/NeedSnapshotCache"),
	}
}

func (nsCache *NeedSnapshotCache) findIndex(blockList []*NeedSnapshotBlock, hashHeight *ledger.HashHeight) (uint64, error) {
	blockListLength := uint64(len(blockList))
	headBlock := blockList[blockListLength-1]
	headBlockHeight := headBlock.Block.Height
	if headBlockHeight < hashHeight.Height {
		err := errors.New(fmt.Sprintf("headBlockHeight < height, headBlockHeight is %d, height is %d", headBlockHeight, hashHeight.Height))
		return 0, err
	}

	gap := headBlockHeight - hashHeight.Height
	if gap > blockListLength-1 {
		err := errors.New(fmt.Sprintf("blockList is too short, length is %d, headBlockHeight is %d, hashHeight.Height is %d",
			len(blockList), headBlockHeight, hashHeight.Height))
		return 0, err
	}

	index := blockListLength - 1 - gap
	if blockList[index].Block.Hash != hashHeight.Hash {
		err := errors.New(fmt.Sprintf("blockList[index].Hash != hashHeight.Hash, blockList[index].Hash is %s, blockList[index].Height is %d, hashHeight.Hash is %s, hashHeight.Height is %d",
			blockList[index].Block.Hash, blockList[index].Block.Height, hashHeight.Hash, hashHeight.Height))
		return 0, err
	}
	return index, nil
}

func (nsCache *NeedSnapshotCache) Build() {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()

	unconfirmedSubLedger, getSubLedgerErr := nsCache.chain.GetUnConfirmedSubLedger()
	if getSubLedgerErr != nil {
		nsCache.log.Crit("GetUnConfirmedSubLedger failed, error is "+getSubLedgerErr.Error(), "method", "Build")
	}

	nsCache.subLedger = make(map[types.Address][]*NeedSnapshotBlock)
	for _, blocks := range unconfirmedSubLedger {
		nsCache.addBlocks(blocks)
	}
}

func (nsCache *NeedSnapshotCache) buildPart(addrList []types.Address) {

	unconfirmedSubLedger, getSubLedgerErr := nsCache.chain.GetUnConfirmedPartSubLedger(addrList)
	if getSubLedgerErr != nil {
		nsCache.log.Crit("GetUnConfirmedSubLedger failed, error is "+getSubLedgerErr.Error(), "method", "Build")
	}

	for _, addr := range addrList {
		delete(nsCache.subLedger, addr)
		nsCache.addBlocks(unconfirmedSubLedger[addr])
	}
}

func (nsCache *NeedSnapshotCache) recalculateQuota(addr types.Address) {
	cacheBlockList := nsCache.subLedger[addr]

	accumulateQuota := uint64(0)
	for _, cacheBlock := range cacheBlockList {
		accumulateQuota += cacheBlock.Block.Quota
		cacheBlock.AccumulateQuota = accumulateQuota
	}
}

func (nsCache *NeedSnapshotCache) addBlocks(blocks []*ledger.AccountBlock) {
	addr := blocks[0].AccountAddress
	accumulateQuota := uint64(0)
	cacheBlockList := nsCache.subLedger[addr]

	// 0 means front, 1 means behind
	insertPosition := 0
	if len(cacheBlockList) > 0 {
		tailBlockHeight := blocks[0].Height
		headCacheBlock := cacheBlockList[len(cacheBlockList)-1]
		if tailBlockHeight == headCacheBlock.Block.Height+1 {
			accumulateQuota = cacheBlockList[len(cacheBlockList)-1].AccumulateQuota
			insertPosition = 1
		}
	}

	needRecalculateQuota := false
	needSnapshotBlocks := make([]*NeedSnapshotBlock, len(blocks))
	for index, block := range blocks {
		if helper.MaxUint64-accumulateQuota >= block.Quota {
			accumulateQuota += block.Quota
		} else {
			needRecalculateQuota = true
		}
		needSnapshotBlocks[index] = &NeedSnapshotBlock{
			Block:           block,
			AccumulateQuota: accumulateQuota,
		}
	}

	if insertPosition == 0 {
		// recalculate quota
		accumulateQuota := needSnapshotBlocks[len(needSnapshotBlocks)-1].AccumulateQuota

		for _, cacheBlock := range cacheBlockList {
			accumulateQuota += cacheBlock.Block.Quota
			cacheBlock.AccumulateQuota = accumulateQuota
		}

		nsCache.subLedger[addr] = append(needSnapshotBlocks, cacheBlockList...)
	} else {
		nsCache.subLedger[addr] = append(cacheBlockList, needSnapshotBlocks...)
	}

	if needRecalculateQuota {
		nsCache.recalculateQuota(addr)
	}
}

func (nsCache *NeedSnapshotCache) addBlock(block *ledger.AccountBlock) {
	accumulateQuota := block.Quota
	addr := block.AccountAddress
	blockList := nsCache.subLedger[addr]
	needRecalculateQuota := false
	if len(blockList) > 0 {
		prevBlock := blockList[len(blockList)-1]
		prevAccumulateQuota := prevBlock.AccumulateQuota

		if helper.MaxUint64-prevAccumulateQuota >= accumulateQuota {
			accumulateQuota += prevAccumulateQuota
		} else {
			needRecalculateQuota = true
		}
	}
	blockList = append(blockList, &NeedSnapshotBlock{
		Block:           block,
		AccumulateQuota: accumulateQuota,
	})
	nsCache.subLedger[addr] = blockList

	if needRecalculateQuota {
		nsCache.recalculateQuota(addr)
	}
}

func (nsCache *NeedSnapshotCache) InsertAccountBlock(block *ledger.AccountBlock) error {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()

	if cacheBlocks, ok := nsCache.subLedger[block.AccountAddress]; ok && len(cacheBlocks) > 0 {
		headCacheBlock := cacheBlocks[len(cacheBlocks)-1]

		if block.Height != headCacheBlock.Block.Height+1 {
			err := errors.New(fmt.Sprintf("block.Height != headBlock.Height+1, addr is %s, block.Height is %d, headBlock.Height is %d",
				block.AccountAddress, block.Height, headCacheBlock.Block.Height))
			nsCache.log.Error(err.Error(), "method", "InsertAccountBlock")
			return err
		}
	}

	nsCache.addBlock(block)
	return nil
}

func (nsCache *NeedSnapshotCache) HasSnapshot(subLedger ledger.SnapshotContent) (uint64, error) {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()

	var processedAddrList []types.Address

	snapshotQuota := uint64(0)
	for addr, hashHeight := range subLedger {
		blockList := nsCache.subLedger[addr]
		if len(blockList) <= 0 {
			err := errors.New(fmt.Sprintf("blockList is nil, addr is %s, block.Height is %d, block.Hash is %s",
				addr, hashHeight.Height, hashHeight.Hash))

			nsCache.log.Error(err.Error(), "method", "HasSnapshot")

			//recover cache
			nsCache.buildPart(processedAddrList)
			return 0, err
		}

		index, err := nsCache.findIndex(blockList, hashHeight)
		if err != nil {
			nsCache.log.Error("findIndex failed, error is "+err.Error(), "method", "HasSnapshot")

			//recover cache
			nsCache.buildPart(processedAddrList)
			return 0, err
		}

		tailBlock := blockList[0]
		snapshotQuota += blockList[index].AccumulateQuota - tailBlock.AccumulateQuota + tailBlock.Block.Quota
		blockList = blockList[index+1:]
		nsCache.subLedger[addr] = blockList

		processedAddrList = append(processedAddrList, addr)
	}
	return snapshotQuota, nil
}

func (nsCache *NeedSnapshotCache) NotSnapshot(horizontalLine map[types.Address]*ledger.AccountBlock) {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()

	for addr, block := range horizontalLine {
		blockList := nsCache.subLedger[addr]
		if len(blockList) <= 0 {
			continue
		}

		index, err := nsCache.findIndex(blockList, &ledger.HashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		})

		if err != nil {
			delete(nsCache.subLedger, addr)
			continue
		}

		blockList = blockList[:index]
		nsCache.subLedger[addr] = blockList
	}
}

func (nsCache *NeedSnapshotCache) GetBlockByHash(addr types.Address, hashHeight *ledger.HashHeight) *ledger.AccountBlock {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()
	blockList := nsCache.subLedger[addr]
	if len(blockList) <= 0 {
		return nil
	}
	index, err := nsCache.findIndex(blockList, hashHeight)
	if err == nil {
		return blockList[index].Block
	}
	return nil

}
func (nsCache *NeedSnapshotCache) NeedReSnapshot(subLedger map[types.Address][]*ledger.AccountBlock) error {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()

	var processedAddrList []types.Address

	for addr, blocks := range subLedger {
		cacheBlockList := nsCache.subLedger[addr]
		if len(cacheBlockList) <= 0 {
			nsCache.addBlocks(blocks)
			continue
		}

		if cacheBlockList[0].Block.Height != blocks[len(blocks)-1].Height+1 {
			err := errors.New(fmt.Sprintf("cacheBlockList[0].Block.Height != blocks[len(blocks)-1].Height, addr is %s, cacheBlockList[0].Block.Height is %d, blocks[len(blocks)-1].Height is %d",
				addr, cacheBlockList[0].Block.Height, blocks[len(blocks)-1].Height))

			nsCache.log.Error(err.Error(), "method", "NeedReSnapshot")

			//recover cache
			nsCache.buildPart(processedAddrList)
			return err
		}

		nsCache.addBlocks(blocks)

		processedAddrList = append(processedAddrList, addr)
	}
	return nil
}

func (nsCache *NeedSnapshotCache) GetSnapshotContent() ledger.SnapshotContent {
	nsCache.lock.Lock()
	defer nsCache.lock.Unlock()

	content := make(ledger.SnapshotContent, 0)
	for addr, blocks := range nsCache.subLedger {
		if len(blocks) <= 0 {
			continue
		}
		headBlock := blocks[len(blocks)-1]
		content[addr] = &ledger.HashHeight{
			Height: headBlock.Block.Height,
			Hash:   headBlock.Block.Hash,
		}
	}
	return content
}
