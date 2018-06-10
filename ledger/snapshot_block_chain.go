package ledger

import (
	"go-vite/vitedb"
	"fmt"
)

type SnapshotBlockChain struct {
	db *vitedb.DataBase
}

func (SnapshotBlockChain) New () *SnapshotBlockChain {
	db:= vitedb.GetDataBase(vitedb.DB_BLOCK)

	return &SnapshotBlockChain{
		db: db,
	}
}

func (sbc * SnapshotBlockChain) WriteBlock (block *SnapshotBlock) error {
	// 模拟key, 需要改
	key :=  []byte("snapshot_test")

	// Block serialize by protocol buffer
	data, err := block.Serialize()

	if err != nil {
		fmt.Println(err)
		return err
	}

	sbc.db.Put(key, data)
	return nil
}

func (sbc * SnapshotBlockChain) GetBlock (key []byte) (*SnapshotBlock, error) {
	block, err := sbc.db.Get(key)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	snapshotBlock := &SnapshotBlock{}

	snapshotBlock.Deserialize(block)

	return snapshotBlock, nil
}