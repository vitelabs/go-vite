package main

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/rand"
	"math/big"
	"time"
	"github.com/vitelabs/go-vite/ledger/access"
	"log"
)

var accountChainAccess = access.GetAccountChainAccess()

func writeGenesisBlocks () {
	blocks := ledger.GetGenesisBlocks()
	err := accountChainAccess.WriteBlockList(blocks)
	if err != nil {
		log.Fatal(err)
	}
}

func writeAccoutChain() ([]*ledger.AccountBlock) {
	var accountChain []*ledger.AccountBlock


	return accountChain
}

func createReceiveBlock(address *types.Address, fromAddress *types.Address, fromBlock *ledger.AccountBlock) *ledger.AccountBlock {
	accountBlock := &ledger.AccountBlock{
		AccountAddress: address,
		Hash: createBlockHash(),
		From: fromAddress,
		FromHash: fromBlock.Hash,

		Amount: fromBlock.Amount,
		Timestamp: uint64(time.Time{}.Unix()),

		TokenId: fromBlock.TokenId,

		Data: "haha" + string(time.Time{}.Unix()),

		Signature: createAccountBlockSignature(),

		Nounce: createNounce(),

		Difficulty: createDifficulty(),

		FAmount: createAmount(big.NewInt(10000)),
	}

	return accountBlock
}

func createSendBlock (sendAddress, toAddress *types.Address) *ledger.AccountBlock{
	accountBlock := &ledger.AccountBlock{
		AccountAddress: sendAddress,
		Hash: createBlockHash(),
		To: toAddress,
		Amount: createAmount(big.NewInt(1000000000)),

		Timestamp: uint64(time.Time{}.Unix()),
		TokenId: &viteTokenId,

		Data: "haha" + string(time.Time{}.Unix()),

		Signature: createAccountBlockSignature(),

		Nounce: createNounce(),

		Difficulty: createDifficulty(),

		FAmount: createAmount(big.NewInt(10000)),
	}

	return accountBlock
}


func createBlockHash () []byte{
	return []byte("IamBlockHash")
}

func createAmount (max *big.Int) *big.Int {
	amount := &big.Int{}
	amount.Rand(&rand.Rand{}, max)

	return amount
}

func createAccountBlockSignature () []byte {
	return []byte("1234567890123456789")
}

func createNounce () []byte {
	return []byte("IamNounce")
}


func createDifficulty () []byte {
	return []byte("IamDifficulty")
}

