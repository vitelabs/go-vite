package service

import (
	"go-vite/ledger"
	"fmt"
)

type AccountChain struct {}

func (*AccountChain) SendBlock (block * ledger.AccountBlock, blockHash *[]byte) error {
	fmt.Print("%+v", *block)
	*blockHash = []byte("123")
	return nil
}

func (*AccountChain) GetBlockByHash (blockHash *[]byte, block * ledger.AccountBlock) error {
	return nil
}

func (*AccountChain) GetBlockList (blockQuery *AccountBlockQuery, blockList []*ledger.AccountBlock) error {
	return nil
}

// response
//func (*Chain) receive () {
//
//}