package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"testing"
)

func TestChain_DeleteSnapshotBlocks(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	chainInstance, accounts, snapshotBlockList := SetUp(t, 18, 96, 2)

	snapshotBlockList = testInsertAndDelete(t, chainInstance, accounts, snapshotBlockList)

	TearDown(chainInstance)
}

func testInsertAndDelete(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	t.Run("DeleteSbsAndAbs", func(t *testing.T) {
		snapshotBlockList = testDeleteSbsAndAbs(t, chainInstance, accounts, snapshotBlockList)
	})

	for i := 0; i < rand.Intn(3)+1; i++ {
		t.Run("DeleteSnapshotBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteSnapshotBlocks(t, chainInstance, accounts, snapshotBlockList, rand.Intn(10))
		})

		t.Run("DeleteAccountBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteAccountBlocks(t, chainInstance, accounts, snapshotBlockList)
		})

	}

	return snapshotBlockList
}

func testDeleteSbsAndAbs(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, 5000, 1)...)
	deleteCount := 3500
	DeleteSnapshotBlocks(t, chainInstance, accounts, uint64(deleteCount))

	snapshotBlockList = snapshotBlockList[:len(snapshotBlockList)-deleteCount]
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList

}

func testDeleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock, deleteCount int) []*ledger.SnapshotBlock {
	insertCount := rand.Intn(1000)
	snapshotPerNum := rand.Intn(20)

	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, insertCount, snapshotPerNum)...)

	if deleteCount > len(snapshotBlockList) {
		lackNum := deleteCount - len(snapshotBlockList) + 10
		snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, lackNum*20, 20)...)
	}

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	//deleteCount := (rand.Uint64() % 9) + 1
	//deleteCount := uint64(2)

	DeleteSnapshotBlocks(t, chainInstance, accounts, uint64(deleteCount))

	snapshotBlockList = snapshotBlockList[:len(snapshotBlockList)-deleteCount]

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList
}

func testDeleteAccountBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {

	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, rand.Intn(1000), 23)...)
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	DeleteAccountBlocks(t, chainInstance, accounts)

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList
}

func DeleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, count uint64) {
	snapshotBlocksToDelete, err := chainInstance.GetSnapshotBlocks(chainInstance.GetLatestSnapshotBlock().Hash, false, count)

	if err != nil {
		t.Fatal(err)
	}
	if _, err := chainInstance.DeleteSnapshotBlocksToHeight(chainInstance.GetLatestSnapshotBlock().Height + 1 - count); err != nil {
		snapshotBlocksStr := ""
		for _, snapshotBlock := range snapshotBlocksToDelete {
			snapshotBlocksStr += fmt.Sprintf("%+v, ", snapshotBlock)
		}

		t.Fatal(fmt.Sprintf("Error: %s, snapshotBlocks: %s", err, snapshotBlocksStr))
	}

	for _, account := range accounts {
		account.DeleteSnapshotBlocks(accounts, snapshotBlocksToDelete)
	}
}

func DeleteAccountBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	unconfirmedBlocks := chainInstance.cache.GetUnconfirmedBlocks()
	if len(unconfirmedBlocks) <= 0 {
		return
	}

	unconfirmedBlock := unconfirmedBlocks[rand.Intn(len(unconfirmedBlocks))]

	account := accounts[unconfirmedBlock.AccountAddress]
	_, err := chainInstance.DeleteAccountBlocks(account.addr, unconfirmedBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}
	//fmt.Println("delete accountBlocks")
	//fmt.Println()
	//for _, accountBlock := range accountBlocksDeleted {
	//	fmt.Printf("%+v\n", accountBlock)
	//	fmt.Println()
	//}
	//fmt.Println("delete accountBlocks end")

	onRoadBlocksCache := make(map[types.Hash]struct{})

	//fmt.Println("delete mem accountBlocks")
	//fmt.Println("")

	deleteMemAccountBlock(accounts, account, unconfirmedBlock, onRoadBlocksCache)

	//fmt.Println("delete mem accountBlocks end")

	for fromBlockHash := range onRoadBlocksCache {
		var onRoadSendBlock *vm_db.VmAccountBlock
		for _, account := range accounts {
			if len(account.SendBlocksMap) <= 0 {
				continue
			}
			if sendBlock, ok := account.SendBlocksMap[fromBlockHash]; ok {
				onRoadSendBlock = sendBlock
				break
			}

		}
		if onRoadSendBlock == nil {
			t.Fatal(fmt.Sprintf("error, %s", fromBlockHash))
		}

		toAccount := accounts[onRoadSendBlock.AccountBlock.ToAddress]
		toAccount.AddOnRoadBlock(onRoadSendBlock)

	}
}

func deleteMemAccountBlock(accounts map[types.Address]*Account, account *Account,
	toBlock *ledger.AccountBlock, onRoadBlocksCache map[types.Hash]struct{}) {

	deleteSendBlock := func(blockToDelete *ledger.AccountBlock) {
		delete(onRoadBlocksCache, blockToDelete.Hash)
		toAccount := accounts[blockToDelete.ToAddress]
		var blockNeedDelete *vm_db.VmAccountBlock
		for _, block := range toAccount.ReceiveBlocksMap {
			if block.AccountBlock.FromBlockHash == blockToDelete.Hash {
				blockNeedDelete = block
			}
		}
		if blockNeedDelete != nil {
			deleteMemAccountBlock(accounts, toAccount, blockNeedDelete.AccountBlock, onRoadBlocksCache)
		}
	}
	for {
		blockToDelete := account.latestBlock
		if blockToDelete == nil || blockToDelete.Height < toBlock.Height {
			break
		}

		if blockToDelete.IsSendBlock() {
			deleteSendBlock(blockToDelete)
		} else {
			onRoadBlocksCache[blockToDelete.FromBlockHash] = struct{}{}
		}
		for _, sendBlock := range blockToDelete.SendBlockList {
			deleteSendBlock(sendBlock)
		}

		account.deleteAccountBlock(accounts, blockToDelete.Hash)
		account.rollbackLatestBlock()

		if blockToDelete.Height <= toBlock.Height {
			break
		}
	}

}
