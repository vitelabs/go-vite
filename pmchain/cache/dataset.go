package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type dataSet struct {
	dataId uint64

	dataRefCount map[uint64]int16

	blockDataId map[types.Hash]uint64

	accountBlockSet map[uint64]*ledger.AccountBlock

	snapshotBlockSet map[uint64]*ledger.SnapshotBlock
}

func NewDataSet() *dataSet {
	return &dataSet{
		accountBlockSet:  make(map[uint64]*ledger.AccountBlock, 0),
		snapshotBlockSet: make(map[uint64]*ledger.SnapshotBlock, 0),
	}
}

func (ds *dataSet) RefDataId(dataId uint64) {
	if refCount, ok := ds.dataRefCount[dataId]; ok {
		ds.dataRefCount[dataId] = refCount + 1
	}
}
func (ds *dataSet) UnRefDataId(dataId uint64) {
	if refCount, ok := ds.dataRefCount[dataId]; ok {
		newRefCount := refCount - 1
		if newRefCount <= 0 {
			ds.gc(dataId)
		} else {
			ds.dataRefCount[dataId] = newRefCount
		}
	}
}

func (ds *dataSet) InsertAccountBlock(accountBlock *ledger.AccountBlock) uint64 {
	if dataId, ok := ds.blockDataId[accountBlock.Hash]; ok {
		return dataId
	}

	newDataId := ds.newDataId()
	ds.accountBlockSet[newDataId] = accountBlock
	ds.blockDataId[accountBlock.Hash] = newDataId
	return newDataId
}

func (ds *dataSet) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) uint64 {
	if dataId, ok := ds.blockDataId[snapshotBlock.Hash]; ok {
		return dataId
	}

	newDataId := ds.newDataId()
	ds.snapshotBlockSet[newDataId] = snapshotBlock
	ds.blockDataId[snapshotBlock.Hash] = newDataId

	return newDataId
}

func (ds *dataSet) GetAccountBlock(dataId uint64) *ledger.AccountBlock {
	return ds.accountBlockSet[dataId]
}

func (ds *dataSet) GetSnapshotBlock(dataId uint64) *ledger.SnapshotBlock {
	return ds.snapshotBlockSet[dataId]
}

func (ds *dataSet) gc(dataId uint64) {

}

func (ds *dataSet) newDataId() uint64 {
	ds.dataId++
	return ds.dataId
}
