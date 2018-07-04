package vitedb

import (
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"math/big"
	"encoding/hex"
)

type SnapshotChain struct {
	db *DataBase
}

var _snapshotChain *SnapshotChain
func GetSnapshotChain () *SnapshotChain {
	if _snapshotChain == nil {
		db, err:= GetLDBDataBase(DB_BLOCK)
		if err != nil {
			log.Fatal(err)
		}

		_snapshotChain = &SnapshotChain{
			db: db,
		}
	}

	return _snapshotChain

}

func (sbc *SnapshotChain) GetHeightByHash (blockHash []byte) (*big.Int, error) {
	key, err:= createKey(DBKP_SNAPSHOTBLOCKHASH, hex.EncodeToString(blockHash))

	heightBytes, err := sbc.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, nil
	}

	height := &big.Int{}
	height.SetBytes(heightBytes)
	return height, nil
}

func (sbc *SnapshotChain) GetBlockByHash (blockHash []byte) (*ledger.SnapshotBlock, error){
	return nil, nil
}

func (sbc *SnapshotChain) GetBlockList (index int, num int, count int) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (sbc * SnapshotChain) WriteBlock (block *ledger.SnapshotBlock) error {
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