package vitedb

import (
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"math/big"
	"encoding/hex"
	"github.com/syndtr/goleveldb/leveldb/util"
	"errors"
)

type SnapshotChain struct {
	db *DataBase
}

var _snapshotChain *SnapshotChain

func GetSnapshotChain() *SnapshotChain {
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

func (sbc *SnapshotChain) GetHeightByHash(blockHash []byte) (*big.Int, error) {
	key, err := createKey(DBKP_SNAPSHOTBLOCKHASH, hex.EncodeToString(blockHash))

	heightBytes, err := sbc.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, nil
	}

	height := &big.Int{}
	height.SetBytes(heightBytes)
	return height, nil
}

func (sbc *SnapshotChain) GetBlockByHash (blockHash []byte) (*ledger.SnapshotBlock, error) {
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
		return nil, errors.New("GetBlockList failed. Cause the SnapshotChain has no block")
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

func (sbc *SnapshotChain) GetLatestBlockHeight () (*big.Int, error) {
	key, ckErr := createKey(DBKP_SNAPSHOTBLOCK,nil)
	if ckErr != nil {
		return nil, ckErr
	}
	iter := sbc.db.Leveldb.NewIterator(util.BytesPrefix(key),nil)
	defer iter.Release()
	if !iter.Last() {
		return nil, errors.New("GetLatestBlockHeight failed.")
	}
	sb := &ledger.SnapshotBlock{}
	sdErr := sb.DbDeserialize(iter.Value())
	if sdErr != nil {
		return nil, sdErr
	}
	return sb.Height, nil
}


func (sbc *SnapshotChain) Iterate (iterateFunc func(snapshotBlock *ledger.SnapshotBlock) bool, startBlockHash []byte) error {
	startHeight, err := sbc.GetHeightByHash(startBlockHash)
	if err != nil {
		return err
	}

	startKey, err := createKey(DBKP_SNAPSHOTBLOCK, startHeight)

	iter := sbc.db.Leveldb.NewIterator(&util.Range{Start: startKey}, nil)
	defer iter.Release()

	for {
		value := iter.Value()
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

func (sbc *SnapshotChain) WriteBlock (block *ledger.SnapshotBlock) error {
	//// 模拟key, 需要改
	//key :=  []byte("snapshot_test")
	//
	//// Block serialize by protocol buffer
	//data, err := block.Serialize()
	//
	//if err != nil {
	//	fmt.Println(err)
	//	return err
	//}
	//
	//sbc.db.Put(key, data)
	return nil
}

func (sbc *SnapshotChain) WriteBlockHeight (block *ledger.SnapshotBlock, ) error {
	return nil
}

