package vitedb

import (
	"github.com/vitelabs/go-vite/ledger"
	"log"
)

type SnapshotChain struct {
	db *DataBase
}

var _snapshotChain *SnapshotChain
func (SnapshotChain) GetInstance () *SnapshotChain {
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

func (sbc * SnapshotChain) GetBlock (key []byte) (*ledger.SnapshotBlock, error) {
	//block, err := sbc.db.Get(key)
	//if err != nil {
	//	fmt.Println(err)
	//	return nil, err
	//}
	//snapshotBlock := &ledger.SnapshotBlock{}
	//
	//snapshotBlock.Deserialize(block)

	return nil, nil
}