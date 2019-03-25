package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type UnconfirmedPool struct {
	ds *dataSet

	insertedList []uint64
	insertedMap  map[types.Address][]uint64
}

func NewUnconfirmedPool(ds *dataSet) *UnconfirmedPool {
	return &UnconfirmedPool{
		ds:          ds,
		insertedMap: make(map[types.Address][]uint64),
	}
}

func (up *UnconfirmedPool) InsertAccountBlock(address *types.Address, dataId uint64) {

	up.insertedList = append(up.insertedList, dataId)
	up.insertedMap[*address] = append(up.insertedMap[*address], dataId)

	up.ds.RefDataId(dataId)
}

// No lock
func (up *UnconfirmedPool) GetBlocks() []*ledger.AccountBlock {
	blocks := make([]*ledger.AccountBlock, 0, len(up.insertedList))
	for _, dataId := range up.insertedList {
		blocks = append(blocks, up.ds.GetAccountBlock(dataId))
	}
	return blocks
}

func (up *UnconfirmedPool) GetBlocksByAddress(addr *types.Address) []*ledger.AccountBlock {

	list := up.insertedMap[*addr]

	blocks := make([]*ledger.AccountBlock, 0, len(list))
	for _, dataId := range list {
		blocks = append(blocks, up.ds.GetAccountBlock(dataId))
	}
	return blocks
}

func (up *UnconfirmedPool) DeleteBlocks(blocks []*ledger.AccountBlock) {

	newInsertedList := make([]uint64, 0, len(up.insertedList)-len(blocks))
	newInsertedMap := make(map[types.Address][]uint64)

	deletedDataIdMap := make(map[uint64]*ledger.AccountBlock, len(blocks))
	for _, block := range blocks {
		dataId := up.ds.GetDataId(&block.Hash)
		deletedDataIdMap[dataId] = block
	}
	for _, insertedDataId := range up.insertedList {
		if block, ok := deletedDataIdMap[insertedDataId]; !ok {
			newInsertedList = append(newInsertedList, insertedDataId)
			newInsertedMap[block.AccountAddress] = append(newInsertedMap[block.AccountAddress], insertedDataId)
		} else {
			// un ref
			up.ds.UnRefDataId(insertedDataId)
		}

	}
	up.insertedList = newInsertedList
	up.insertedMap = newInsertedMap
}

func (up *UnconfirmedPool) DeleteAllBlocks() {
	for _, insertedDataId := range up.insertedList {
		// query
		deleteBlock := up.ds.GetAccountBlock(insertedDataId)
		// un ref
		up.ds.UnRefDataId(insertedDataId)

		for _, sendBlock := range deleteBlock.SendBlockList {
			up.ds.UnRefByBlockHash(&sendBlock.Hash)
		}

	}
	up.insertedList = nil
	up.insertedMap = make(map[types.Address][]uint64)
}
