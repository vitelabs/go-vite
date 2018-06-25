package vitedb

import (
	"fmt"
	"go-vite/ledger"
)

type AccountBlockChain struct {
	db *DataBase
	accountStore *Account
}

func (bc AccountBlockChain) New () *AccountBlockChain {
	db := GetDataBase(DB_BLOCK)
	return &AccountBlockChain{
		db: db,
		accountStore: Account{}.New(),
	}
}

func (bc * AccountBlockChain) WriteBlock (block *ledger.AccountBlock) error {
	accountMeta := bc.accountStore.GetAccountMeta(block.AccountAddress)

	if accountMeta == nil {
	}


	if block.FromHash == nil {
		// It is send block

	} else {
		// It is receive block

	}

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

func (bc * AccountBlockChain) GetBlock (key []byte) (*ledger.AccountBlock, error) {
	block, err := bc.db.Get(key)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	accountBlock := &ledger.AccountBlock{}
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