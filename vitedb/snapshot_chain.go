package vitedb

import (
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"math/big"
	"github.com/syndtr/goleveldb/leveldb/util"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
)

type SnapshotChain struct {
	db *DataBase
}

var _snapshotChain *SnapshotChain

func GetSnapshotChain () *SnapshotChain {
	if _snapshotChain == nil {
		db, err := GetLDBDataBase(DB_BLOCK)
		if err != nil {
			log.Fatal(err)
		}

		_snapshotChain = &SnapshotChain{
			db: db,
		}
	}

	return _snapshotChain

}

func (sbc *SnapshotChain) GetHeightByHash (blockHash *types.Hash) (*big.Int, error) {
	key, err := createKey(DBKP_SNAPSHOTBLOCKHASH, blockHash.Bytes())

	heightBytes, err := sbc.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, nil
	}

	height := &big.Int{}
	height.SetBytes(heightBytes)
	return height, nil
}

func (sbc *SnapshotChain) GetBlockByHash (blockHash *types.Hash) (*ledger.SnapshotBlock, error) {
	blockHeight, ghErr := sbc.GetHeightByHash(blockHash)
	if ghErr != nil {
		return nil, ghErr
	}
	snapshotBlcok, gbErr := sbc.GetBLockByHeight(blockHeight)
	if gbErr != nil {
		return nil, gbErr
	}
	return snapshotBlcok, nil
}

func (sbc *SnapshotChain) GetBlockList (index, num, count int) ([]*ledger.SnapshotBlock, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, nil)
	if ckErr != nil {
		return nil, ckErr
	}
	iter := sbc.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	var blockList []*ledger.SnapshotBlock
	if !iter.Last() {
		return nil, errors.New("GetBlockList failed. Cause the SnapshotChain has no block" + iter.Error().Error())
	}
	for i := 0; i < (index+num)*count; i++ {
		if i >= index*count {
			snapshotBLock := &ledger.SnapshotBlock{}
			dsErr := snapshotBLock.DbDeserialize(iter.Value())
			if dsErr != nil {
				return nil, dsErr
			}
			blockList = append(blockList, snapshotBLock)
		}
		if !iter.Prev() {
			if err := iter.Error(); err != nil {
				return nil, err
			}
			break
		}
	}
	return blockList, nil
}

func (sbc * SnapshotChain) GetBLockByHeight (blockHeight *big.Int) (*ledger.SnapshotBlock, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, blockHeight)
	if ckErr != nil {
		return nil, ckErr
	}
	data, dbErr := sbc.db.Leveldb.Get(key,nil)
	if dbErr != nil {
		return nil, dbErr
	}
	sb := &ledger.SnapshotBlock{}
	dsErr := sb.DbDeserialize(data)
	if dsErr != nil {
		return nil, dsErr
	}
	return sb, nil
}

func (sbc *SnapshotChain) GetBlocksFromOrigin (originBlockHash *types.Hash, count uint64, forward bool) ([]*ledger.SnapshotBlock, error) {
	originBlock, err := sbc.GetBlockByHash(originBlockHash)

	if err != nil {
		return nil, err
	}


	var startHeight, endHeight, gap = &big.Int{}, &big.Int{}, &big.Int{}
	gap.SetUint64(count)

	if forward {
		startHeight = originBlock.Height
		endHeight.Add(startHeight, gap)
	} else {
		endHeight = originBlock.Height
		startHeight.Sub(endHeight, gap)
	}

	startKey, err := createKey(DBKP_SNAPSHOTBLOCK, startHeight)
	if err != nil {
		return nil, err
	}

	limitKey, err := createKey(DBKP_SNAPSHOTBLOCK, endHeight)
	if err != nil {
		return nil, err
	}

	iter := sbc.db.Leveldb.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()


	var sbList []*ledger.SnapshotBlock

	for iter.Next() {
		sb := &ledger.SnapshotBlock{}
		sdErr := sb.DbDeserialize(iter.Value())
		if sdErr != nil {
			return nil, sdErr
		}

		sbList = append(sbList, sb)
	}

	return sbList, nil
}

func (sbc *SnapshotChain) GetLatestBlock () (*ledger.SnapshotBlock, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, nil)
	if ckErr != nil {
		return nil, ckErr
	}
	iter := sbc.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	if !iter.Last() {
		return nil, errors.New("GetLatestBlock failed. Cause the SnapshotChain has no block" + iter.Error().Error())
	}
	sb := &ledger.SnapshotBlock{}
	sdErr := sb.DbDeserialize(iter.Value())
	if sdErr != nil {
		return nil, sdErr
	}
	return sb, nil
}


func (sbc *SnapshotChain) Iterate (iterateFunc func(snapshotBlock *ledger.SnapshotBlock) bool, startBlockHash *types.Hash) error {
	startHeight, err := sbc.GetHeightByHash(startBlockHash)
	if err != nil {
		return err
	}

	if startHeight == nil {
		return nil
	}

	startKey, err := createKey(DBKP_SNAPSHOTBLOCK, startHeight)

	iter := sbc.db.Leveldb.NewIterator(&util.Range{Start: startKey}, nil)
	defer iter.Release()

	for {
		value := iter.Value()
		if value == nil {
			return nil
		}

		var snapshotBlock *ledger.SnapshotBlock
		snapshotBlock.DbDeserialize(value)
		if !iterateFunc(snapshotBlock) {
			return nil
		}

		if !iter.Next() {
			break
		}
	}

	return nil
}

func (sbc *SnapshotChain) WriteBlock (batch *leveldb.Batch, block *ledger.SnapshotBlock) error {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK, block.Height)
	if ckErr != nil {
		return ckErr
	}
	data, sErr := block.DbSerialize()
	if sErr != nil {
		fmt.Println("SnapshotBlock DbSerialize error.")
		return sErr
	}
	batch.Put(key, data)
	return nil
}

func (sbc *SnapshotChain) WriteBlockHeight (batch *leveldb.Batch, block *ledger.SnapshotBlock, ) error {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCKHASH, block.Hash)
	if ckErr != nil {
		return ckErr
	}
	batch.Put(key, block.Height.Bytes())
	return nil
}

func (sbc *SnapshotChain) BatchWrite (batch *leveldb.Batch, writeFunc func (batch *leveldb.Batch) error) error {
	return batchWrite(batch, sbc.db.Leveldb, func (context *batchContext) error {
		return writeFunc(context.Batch)
	})
}

func (sbc *SnapshotChain) GetAccountList () ([]*types.Address, error){
	key, ckErr := createKey(DBKP_ACCOUNTID_INDEX, nil)
	if ckErr != nil {
		return nil, ckErr
	}
	iter := sbc.db.Leveldb.NewIterator(util.BytesPrefix(key),nil)
	defer iter.Release()
	if itErr := iter.Error(); itErr != nil {
		return nil, itErr
	}
	var accountList []*types.Address
	for iter.Next() {
		address, err := types.BytesToAddress(iter.Value())
		if err != nil {
			return nil, err
		}
		accountList = append(accountList, &address)
	}
	return accountList, nil
}