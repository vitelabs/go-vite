package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
)

type indexes struct {
	ds *dataSet

	hashIndexes   map[types.Hash]uint64
	heightIndexes map[types.Address]map[uint64]uint64
}

func NewIndexes(ds *dataSet) *indexes {
	return &indexes{
		ds: ds,

		hashIndexes:   make(map[types.Hash]uint64),
		heightIndexes: make(map[types.Address]map[uint64]uint64),
	}
}

func (index *indexes) InsertAccountBlock(dataId uint64) {
	ab := index.ds.GetAccountBlock(dataId)
	index.hashIndexes[ab.Hash] = dataId

	heightMap := index.heightIndexes[ab.AccountAddress]
	if heightMap == nil {
		heightMap = make(map[uint64]uint64)
	}
	heightMap[ab.Height] = dataId

	index.heightIndexes[ab.AccountAddress] = heightMap
	index.ds.RefDataId(dataId)
}

func (index *indexes) DeleteAccountBlocks(dataIdList []uint64) {
	for _, dataId := range dataIdList {
		index.ds.UnRefDataId(dataId)
	}
}

func (index *indexes) GetAccountBlockByHash(blockHash *types.Hash) uint64 {
	return index.hashIndexes[*blockHash]
}

func (index *indexes) GetAccountBlockByHeight(address *types.Address, height uint64) uint64 {
	return index.heightIndexes[*address][height]
}
