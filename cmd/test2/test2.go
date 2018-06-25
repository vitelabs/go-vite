package main

import (
	"go-vite/vitedb"
	"go-vite/ledger"
)


func createAccountBlock () *ledger.AccountBlock {
	return &ledger.AccountBlock{
		AccountAddress: []byte{1, 2, 3},

		To: []byte{4, 5, 6},

		PrevHash: []byte{7, 8, 9},

		Signature: []byte{19, 20, 21},
	}
}

func main () {
	accountBlockChain := vitedb.AccountBlockChain{}.New()
	accountBlockChain.WriteBlock(createAccountBlock())
}
