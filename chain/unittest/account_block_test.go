package chain_unittest

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func Test_ReceiveHeights(t *testing.T) {
	const PRINT_PER_ACCOUNTS = 1000
	chainInstance := NewChainInstance("ledger_test/testdata", false)

	//latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	snapshotHeight := uint64(3985470)
	allLatestAccountBlock, err := chainInstance.GetAllLatestAccountBlock()
	if err != nil {
		panic(err)
	}
	//allAccounts := make([]*ledger.Account, 0, len(allLatestAccountBlock))

	allOnRoadBlocks := make(map[types.Address][]*ledger.AccountBlock)

	accountNum := 0
	sendBlocksMap := make(map[types.Hash]*ledger.AccountBlock)
	receiveBlocksMap := make(map[types.Hash]*ledger.AccountBlock)
	for _, latestAccountBlock := range allLatestAccountBlock {
		accountNum++
		addr := latestAccountBlock.AccountAddress
		//account, err := chainInstance.GetAccount(&addr)
		//if err != nil {
		//	panic(err)
		//}
		//
		//allAccounts = append(allAccounts, account)
		if onRoadBlocks, err := chainInstance.GetOnRoadBlocksBySendAccount(&addr, snapshotHeight); err != nil {
			panic(err)
		} else if len(onRoadBlocks) > 0 {
			allOnRoadBlocks[addr] = onRoadBlocks
		}

		if sendBlocks, receiveBlocks, err := chainInstance.GetSendAndReceiveBlocks(&addr, snapshotHeight); err != nil {
			panic(err)
		} else {
			for _, sendBlock := range sendBlocks {
				sendBlocksMap[sendBlock.Hash] = sendBlock
			}
			for _, receiveBlock := range receiveBlocks {

				receiveBlocksMap[receiveBlock.Hash] = receiveBlock

			}

		}

		if accountNum%PRINT_PER_ACCOUNTS == 0 || accountNum == len(allLatestAccountBlock) {
			fmt.Printf("Has query %d accounts(total %d accounts)\n", accountNum, len(allLatestAccountBlock))
		}
	}

	for hash, receiveBlock := range receiveBlocksMap {
		if chainInstance.IsGenesisAccountBlock(receiveBlock) {
			continue
		}
		if sendBlock, ok := sendBlocksMap[receiveBlock.FromBlockHash]; !ok {
			err := errors.New(fmt.Sprintf("send block is nil, receive block hash is %s, from block hash is %s",
				receiveBlock.Hash, receiveBlock.FromBlockHash))
			fmt.Printf(err.Error() + "\n")
		} else {
			receiveBlock.Amount = sendBlock.Amount
			receiveBlock.TokenId = sendBlock.TokenId
		}
		receiveBlocksMap[hash] = receiveBlock

	}

	vcpTokenId, err := types.HexToTokenTypeId("tti_251a3e67a41b5ea2373936c8")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("======Test result======\n")
	totalViteAmount := big.NewInt(0)
	for addr, blocks := range allOnRoadBlocks {
		addrTotalViteAmount := big.NewInt(0)
		addrTotalVcpAmount := big.NewInt(0)
		for _, block := range blocks {
			if block.TokenId == ledger.ViteTokenId {
				addrTotalViteAmount.Add(addrTotalViteAmount, block.Amount)
				totalViteAmount.Add(totalViteAmount, block.Amount)
			} else if block.TokenId == vcpTokenId {
				addrTotalVcpAmount.Add(addrTotalVcpAmount, block.Amount)
			}
		}
		fmt.Printf("%s: %d, %s vite, %s vcp\n", addr, len(blocks), addrTotalViteAmount.String(), addrTotalVcpAmount.String())
	}
	fmt.Printf("total vite: %s\n", totalViteAmount.String())
	fmt.Printf("======Test result======\n")

	totalSendAmount := big.NewInt(0)
	totalReceiveAmount := big.NewInt(0)
	totalOnRoadAmount := big.NewInt(0)
	for _, sendBlock := range sendBlocksMap {
		if sendBlock.TokenId == ledger.ViteTokenId {
			totalSendAmount.Add(totalSendAmount, sendBlock.Amount)
		}
	}

	for _, receiveBlock := range receiveBlocksMap {
		if receiveBlock.TokenId == ledger.ViteTokenId {

			totalReceiveAmount.Add(totalReceiveAmount, receiveBlock.Amount)
		}

	}
	totalOnRoadAmount.Sub(totalSendAmount, totalReceiveAmount)

	fmt.Printf("total send amount is %d\n", totalSendAmount)
	fmt.Printf("total receive amount is %d\n", totalReceiveAmount)
	fmt.Printf("total on road amount is %d\n", totalOnRoadAmount)

	fmt.Printf("query send and receive blocks.\n")

}
