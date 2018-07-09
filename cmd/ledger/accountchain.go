package main

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"time"
	"github.com/vitelabs/go-vite/ledger/access"
	"log"
)

var accountChainAccess = access.GetAccountChainAccess()

var accountAddress1, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10})
var accountAddress2, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11})
var accountAddress3, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 12})
var accountAddress4, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 13})
var accountAddress5, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 14})

func writeGenesisBlocks () {
	blocks := ledger.GetGenesisBlocks()
	err := accountChainAccess.WriteBlockList(blocks)
	if err != nil {
		log.Fatal(err)
	}
}

func writeAccoutChain() {
	var accountChain []*ledger.AccountBlock

	firstPrevHash := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	abs1 := createSendBlock(firstPrevHash, &ledger.GenesisAccount, &accountAddress1, 1000)
	abr1 := createReceiveBlock(nil, &accountAddress1, &ledger.GenesisAccount, abs1)
	//
	abs2 := createSendBlock(abs1.Hash, &ledger.GenesisAccount, &accountAddress2, 2000)
	abs3 := createSendBlock(abs2.Hash, &ledger.GenesisAccount, &accountAddress3, 3000)
	//
	abs4 := createSendBlock(abs3.Hash, &ledger.GenesisAccount, &accountAddress4, 4000)
	abr2 := createReceiveBlock(nil, &accountAddress4, &ledger.GenesisAccount, abs4)
	//
	abs5 := createSendBlock(abs4.Hash, &ledger.GenesisAccount, &accountAddress5, 5000)
	abr3 := createReceiveBlock(nil, &accountAddress5, &ledger.GenesisAccount, abs5)

	//
	abs6 := createSendBlock(abr3.Hash, &accountAddress5, &accountAddress1, 2000)
	abr4 := createReceiveBlock(abr1.Hash, &accountAddress1, &accountAddress5, abs6)

	abr5 := createReceiveBlock(nil, &accountAddress3, &ledger.GenesisAccount, abs3)
	//
	abs7 := createSendBlock(abr5.Hash, &accountAddress3, &accountAddress2, 1000)
	abs8 := createSendBlock(abr2.Hash, &accountAddress4, &accountAddress5, 3000)
	//
	accountChain = []*ledger.AccountBlock{
		abs1,
		abr1,
		abs2,
		abs3,
		abs4,
		abr2,
		abs5,
		abr3,
		abs6,
		abr4,
		abr5,
		abs7,
		abs8,
	}
	err := accountChainAccess.WriteBlockList(accountChain)
	if err != nil {
		log.Fatal(err)
	}
}


func createReceiveBlock(prevHash []byte, address *types.Address, fromAddress *types.Address, fromBlock *ledger.AccountBlock) *ledger.AccountBlock {
	accountBlock := &ledger.AccountBlock{
		AccountAddress: address,
		Hash: createBlockHash(),
		PrevHash: prevHash,

		From: fromAddress,
		FromHash: fromBlock.Hash,

		Amount: fromBlock.Amount,
		Timestamp: uint64(time.Time{}.Unix()),

		TokenId: fromBlock.TokenId,

		Data: "haha" + string(time.Time{}.Unix()),

		Signature: createAccountBlockSignature(),

		Nounce: createNounce(),

		SnapshotTimestamp: ledger.GenesisSnapshotBlockHash,
		Difficulty: createDifficulty(),

		FAmount: big.NewInt(30),
	}

	return accountBlock
}

func createSendBlock (prevHash []byte, sendAddress, toAddress *types.Address, amout int64) *ledger.AccountBlock{
	accountBlock := &ledger.AccountBlock{
		AccountAddress: sendAddress,
		Hash: createBlockHash(),

		PrevHash: prevHash,
		To: toAddress,
		Amount: big.NewInt(amout),

		Timestamp: uint64(time.Time{}.Unix()),
		TokenId: &ledger.MockViteTokenId,

		Data: "haha" + string(time.Time{}.Unix()),

		Signature: createAccountBlockSignature(),

		SnapshotTimestamp: ledger.GenesisSnapshotBlockHash,
		Nounce: createNounce(),

		Difficulty: createDifficulty(),

		FAmount: big.NewInt(20),
	}

	return accountBlock
}


var blochHashCount = 1
func createBlockHash () []byte{
	blochHashCount++
	return []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(blochHashCount)}
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

