package vitedb

import (
	"go-vite/ledger"
)

type SnapshotBlockChain struct {
	db *DataBase
}

func (SnapshotBlockChain) New () *SnapshotBlockChain {
	db:= GetDataBase(DB_BLOCK)

	return &SnapshotBlockChain{
		db: db,
	}
}

func (sbc * SnapshotBlockChain) WriteBlock (block *ledger.SnapshotBlock) error {
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

func (sbc * SnapshotBlockChain) GetBlock (key []byte) (*ledger.SnapshotBlock, error) {
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