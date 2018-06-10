package ledger

import (
	"go-vite/vitedb"
	"fmt"
)

type AccountBlockChain struct {
	db *vitedb.DataBase
}

func (bc AccountBlockChain) New () *AccountBlockChain {
	db := vitedb.GetDataBase(vitedb.DB_BLOCK)
	return &AccountBlockChain{
		db: db,
	}
}

func (bc * AccountBlockChain) WriteBlock (block *AccountBlock) error {
	// 模拟key, 需要改
	key :=  []byte("test")

	// Block serialize by protocol buffer
	data, err := block.Serialize()

	if err != nil {
		fmt.Println(err)
		return err
	}

	bc.db.Put(key, data)
	return nil
}

func (bc * AccountBlockChain) GetBlock (key []byte) (*AccountBlock, error) {
	block, err := bc.db.Get(key)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	accountBlock := &AccountBlock{}
	accountBlock.Deserialize(block)

	return accountBlock, nil
}

func (bc * AccountBlockChain) GetBlockNumberByHash (account []byte, hash []byte) {

}

func (bc * AccountBlockChain) GetBlockByNumber () {

}

func (bc * AccountBlockChain) Iterate (account []byte, startHash []byte, endHash []byte) {

}

func (bc * AccountBlockChain) RevertIterate (account []byte, startHash []byte, endHash []byte) {

}