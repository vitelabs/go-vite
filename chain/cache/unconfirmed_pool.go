package chain_cache

import (
	"github.com/emirpasic/gods/sets/linkedhashset"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type UnconfirmedPool struct {
	ds *dataSet

	pool         *linkedhashset.Set
	insertedList []types.Hash
	insertedMap  map[types.Address][]types.Hash
}

func NewUnconfirmedPool(ds *dataSet) *UnconfirmedPool {
	return &UnconfirmedPool{
		ds:          ds,
		insertedMap: make(map[types.Address][]types.Hash),
	}
}

func (up *UnconfirmedPool) InsertAccountBlock(block *ledger.AccountBlock) {
	up.insertedList = append(up.insertedList, block.Hash)

	up.insertedMap[block.AccountAddress] = append(up.insertedMap[block.AccountAddress], block.Hash)
}

// No lock
func (up *UnconfirmedPool) GetBlocks() []*ledger.AccountBlock {
	blocks := make([]*ledger.AccountBlock, 0, len(up.insertedList))
	for _, hash := range up.insertedList {
		blocks = append(blocks, up.ds.GetAccountBlock(hash))
	}
	return blocks
}

func (up *UnconfirmedPool) GetBlocksByAddress(addr *types.Address) []*ledger.AccountBlock {

	list := up.insertedMap[*addr]

	blocks := make([]*ledger.AccountBlock, 0, len(list))
	for _, hash := range list {
		blocks = append(blocks, up.ds.GetAccountBlock(hash))
	}
	return blocks
}

func (up *UnconfirmedPool) DeleteBlocks(blocks []*ledger.AccountBlock) {
	if len(blocks) <= 0 {
		return
	}

	newInsertedList := make([]types.Hash, 0, len(up.insertedList)-len(blocks))
	newInsertedMap := make(map[types.Address][]types.Hash)

	hashMapToDelete := make(map[types.Hash]struct{}, len(blocks))
	for _, block := range blocks {
		hashMapToDelete[block.Hash] = struct{}{}
	}
	for _, hash := range up.insertedList {
		if _, ok := hashMapToDelete[hash]; !ok {
			newInsertedList = append(newInsertedList, hash)
			block := up.ds.GetAccountBlock(hash)
			newInsertedMap[block.AccountAddress] = append(newInsertedMap[block.AccountAddress], hash)
		}

	}
	up.insertedList = newInsertedList
	up.insertedMap = newInsertedMap
}

func (up *UnconfirmedPool) DeleteAllBlocks() {
	up.insertedList = nil
	up.insertedMap = make(map[types.Address][]types.Hash)
}

//type UnconfirmedPool struct {
//	ds *dataSet
//
//	insertedList []uint64
//	insertedMap  map[types.Address][]uint64
//}
//
//func NewUnconfirmedPool(ds *dataSet) *UnconfirmedPool {
//	return &UnconfirmedPool{
//		ds:          ds,
//		insertedMap: make(map[types.Address][]uint64),
//	}
//}
//
//func (up *UnconfirmedPool) InsertAccountBlock(address *types.Address, dataId uint64) {
//
//	up.insertedList = append(up.insertedList, dataId)
//	up.insertedMap[*address] = append(up.insertedMap[*address], dataId)
//
//	up.ds.RefDataId(dataId)
//}
//
//// No lock
//func (up *UnconfirmedPool) GetBlocks() []*ledger.AccountBlock {
//	blocks := make([]*ledger.AccountBlock, 0, len(up.insertedList))
//	for _, dataId := range up.insertedList {
//		blocks = append(blocks, up.ds.GetAccountBlock(dataId))
//	}
//	return blocks
//}
//
//func (up *UnconfirmedPool) GetBlocksByAddress(addr *types.Address) []*ledger.AccountBlock {
//
//	list := up.insertedMap[*addr]
//
//	blocks := make([]*ledger.AccountBlock, 0, len(list))
//	for _, dataId := range list {
//		blocks = append(blocks, up.ds.GetAccountBlock(dataId))
//	}
//	return blocks
//}
//
//func (up *UnconfirmedPool) DeleteBlocks(blocks []*ledger.AccountBlock) {
//	if len(blocks) <= 0 {
//		return
//	}
//
//	newInsertedList := make([]uint64, 0, len(up.insertedList)-len(blocks))
//	newInsertedMap := make(map[types.Address][]uint64)
//
//	deletedDataIdMap := make(map[uint64]bool, len(blocks))
//	for _, block := range blocks {
//		dataId := up.ds.GetDataId(&block.Hash)
//		deletedDataIdMap[dataId] = true
//	}
//	for _, insertedDataId := range up.insertedList {
//		if _, ok := deletedDataIdMap[insertedDataId]; ok {
//			// un ref
//			up.ds.UnRefDataId(insertedDataId)
//		} else {
//			block := up.ds.GetAccountBlock(insertedDataId)
//			newInsertedList = append(newInsertedList, insertedDataId)
//			newInsertedMap[block.AccountAddress] = append(newInsertedMap[block.AccountAddress], insertedDataId)
//		}
//
//	}
//	up.insertedList = newInsertedList
//	up.insertedMap = newInsertedMap
//}
//
//func (up *UnconfirmedPool) DeleteAllBlocks() {
//	for _, insertedDataId := range up.insertedList {
//		// query
//		deleteBlock := up.ds.GetAccountBlock(insertedDataId)
//		// un ref
//		up.ds.UnRefDataId(insertedDataId)
//
//		for _, sendBlock := range deleteBlock.SendBlockList {
//			up.ds.UnRefByBlockHash(&sendBlock.Hash)
//		}
//
//	}
//	up.insertedList = nil
//	up.insertedMap = make(map[types.Address][]uint64)
//}
