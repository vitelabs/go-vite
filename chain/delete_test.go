package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
	for i := 0; i < 1; i++ {
		chainInstance, accounts, snapshotBlockList := SetUp(50, 960, 1)

		snapshotBlockList = testInsertAndDelete(t, chainInstance, accounts, snapshotBlockList)

		TearDown(chainInstance)
	}

}

func testInsertAndDelete(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	t.Run("DeleteMany", func(t *testing.T) {
		snapshotBlockList = testDeleteMany(t, chainInstance, accounts, snapshotBlockList)
	})

	for i := 0; i < 1; i++ {
		t.Run("deleteSnapshotBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteSnapshotBlocks(t, chainInstance, accounts, snapshotBlockList, rand.Intn(8))
		})

		t.Run("deleteAccountBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteAccountBlocks(t, chainInstance, accounts, snapshotBlockList)
		})

		t.Run("deleteSnapshotBlocks", func(t *testing.T) {
			snapshotBlockList = testDeleteSnapshotBlocks(t, chainInstance, accounts, snapshotBlockList, rand.Intn(8))
		})
	}

	//testRedo(t, chainInstance)

	return snapshotBlockList
}

func testDeleteMany(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, 15000, 3, false)...)
	deleteCount := 3500

	deleteSnapshotBlocks(chainInstance, accounts, uint64(deleteCount))
	chainInstance.stateDB.Store().CompactRange(*util.BytesPrefix([]byte{chain_utils.StorageHistoryKeyPrefix}))

	snapshotBlockList = snapshotBlockList[:len(snapshotBlockList)-deleteCount]

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList

}

func testDeleteSnapshotBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock, deleteCount int) []*ledger.SnapshotBlock {
	insertCount := rand.Intn(10000)
	snapshotPerNum := rand.Intn(200)

	snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, insertCount, snapshotPerNum, false)...)

	if deleteCount > len(snapshotBlockList) {
		lackNum := deleteCount - len(snapshotBlockList) + 10
		snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, lackNum*20, 20, false)...)
	}

	testChainAll(t, chainInstance, accounts, snapshotBlockList)
	//GetLatestAccountBlock(chainInstance, accounts)

	deleteSnapshotBlocks(chainInstance, accounts, uint64(deleteCount))

	snapshotBlockList = snapshotBlockList[:len(snapshotBlockList)-deleteCount]

	//GetLatestAccountBlock(chainInstance, accounts)
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList
}

func testDeleteAccountBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	//snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, rand.Intn(100), 5, false)...)
	//
	//testChainAll(t, chainInstance, accounts, snapshotBlockList)

	deleteAccountBlocks(chainInstance, accounts)

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	return snapshotBlockList
}

// check blocks is a chain
func printf(blocks []*ledger.AccountBlock) string {
	result := ""
	for _, v := range blocks {
		result += fmt.Sprintf("[%d-%s-%s]", v.Height, v.Hash, v.PrevHash)
	}
	return result
}

func deleteSnapshotBlocks(chainInstance *chain, accounts map[types.Address]*Account, count uint64) {
	if count <= 0 {
		return
	}
	snapshotBlocksToDelete, err := chainInstance.GetSnapshotBlocks(chainInstance.GetLatestSnapshotBlock().Hash, false, count)

	if err != nil {
		panic(err)
	}

	snapshotChunksDeleted, err := chainInstance.DeleteSnapshotBlocksToHeight(snapshotBlocksToDelete[len(snapshotBlocksToDelete)-1].Height)
	if err != nil {
		chunksStr := ""
		for _, chunk := range snapshotChunksDeleted {
			if chunk.SnapshotBlock != nil {
				chunksStr += fmt.Sprintf("delete snapshot block %d %s ", chunk.SnapshotBlock.Height, chunk.SnapshotBlock.Hash)
			}
			for _, accountBlock := range chunk.AccountBlocks {
				chunksStr += fmt.Sprintf("delete account block %s %d %s ", accountBlock.AccountAddress, accountBlock.Height, accountBlock.Hash)

			}
		}

		panic(fmt.Sprintf("Error: %s, detail: \n %s", err, chunksStr))
	}

	if len(snapshotBlocksToDelete) != len(snapshotChunksDeleted) &&
		len(snapshotBlocksToDelete)+1 != len(snapshotChunksDeleted) {

		panic(fmt.Sprintf("snapshotBlocksToDelete length: %d, snapshotChunksDeleted length: %d", len(snapshotBlocksToDelete), len(snapshotChunksDeleted)))
	}

	deletedBlocks := make(map[types.Address][]*ledger.AccountBlock)
	for _, chunk := range snapshotChunksDeleted {
		for _, accountBlock := range chunk.AccountBlocks {
			deletedBlocks[accountBlock.AccountAddress] = append(deletedBlocks[accountBlock.AccountAddress], accountBlock)
		}
	}

	for addr, blocks := range deletedBlocks {
		var prev *ledger.AccountBlock
		for _, block := range blocks {
			if prev == nil {
				prev = block
				continue
			}
			if block.PrevHash != prev.Hash {
				panic(fmt.Sprintf("%s-%d %s-%d", prev.Hash, prev.Height, block.Hash, block.Height))

				//panic(errors.New(fmt.Sprintf("%s not a chain:"+printf(blocks), addr)))
			}

			if block.Height-1 != prev.Height {
				panic(errors.New(fmt.Sprintf("%s not a chain:"+printf(blocks), addr)))
			}
			prev = block
		}
	}

	hasStorageRedoLog := len(snapshotChunksDeleted[0].AccountBlocks) <= 0
	for _, account := range accounts {
		account.DeleteSnapshotBlocks(accounts, snapshotBlocksToDelete, hasStorageRedoLog)
	}
}

func deleteAccountBlocks(chainInstance *chain, accounts map[types.Address]*Account) {
	unconfirmedBlocks := chainInstance.cache.GetUnconfirmedBlocks()
	if len(unconfirmedBlocks) <= 0 {
		return
	}

	unconfirmedBlock := unconfirmedBlocks[rand.Intn(len(unconfirmedBlocks))]

	account := accounts[unconfirmedBlock.AccountAddress]
	deletedAccountBlocks, err := chainInstance.DeleteAccountBlocks(account.Addr, unconfirmedBlock.Hash)
	if err != nil {
		panic(err)
	}

	//for i := len(deletedAccountBlocks) - 1; i >= 0; i-- {
	//	ab := deletedAccountBlocks[i]
	//	fmt.Printf("test delete by ab %s %d %s\n", ab.AccountAddress, ab.Height, ab.Hash)
	//
	//	accounts[ab.AccountAddress].deleteAccountBlock(accounts, deletedAccountBlocks[i].Hash)
	//	accounts[ab.AccountAddress].rollbackLatestBlock()
	//
	//}

	onRoadBlocksCache := make(map[types.Hash]struct{})

	deletedHashMap := make(map[types.Hash]struct{}, len(deletedAccountBlocks))
	deleteMemAccountBlock(accounts, account, unconfirmedBlock, onRoadBlocksCache, deletedHashMap)

	for fromBlockHash := range onRoadBlocksCache {
		var onRoadSendBlock *ledger.AccountBlock
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
			continue
		}

		toAccount := accounts[onRoadSendBlock.ToAddress]
		toAccount.AddOnRoadBlock(onRoadSendBlock)
	}

	if len(deletedHashMap) != len(deletedAccountBlocks) {
		panic("error")
	}
	for _, deletedBlock := range deletedAccountBlocks {
		if _, ok := deletedHashMap[deletedBlock.Hash]; !ok {
			panic("error")
		}
	}
}

func deleteMemAccountBlock(accounts map[types.Address]*Account, account *Account, toBlock *ledger.AccountBlock,
	onRoadBlocksCache map[types.Hash]struct{}, deletedHashMap map[types.Hash]struct{}) {

	deleteSendBlock := func(sendBlock *ledger.AccountBlock) {
		delete(onRoadBlocksCache, sendBlock.Hash)
		toAccount := accounts[sendBlock.ToAddress]
		var blockNeedDelete *ledger.AccountBlock
		for _, block := range toAccount.ReceiveBlocksMap {
			if block.FromBlockHash == sendBlock.Hash {
				blockNeedDelete = block
			}
		}
		if blockNeedDelete != nil {
			deleteMemAccountBlock(accounts, toAccount, blockNeedDelete, onRoadBlocksCache, deletedHashMap)
		} else {
			toAccount.DeleteOnRoad(sendBlock.Hash)
		}
	}

	for {
		blockToDelete := account.LatestBlock
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

		//fmt.Println("MEM DELETE ACCOUNT BLOCK", account.Addr, blockToDelete.Height, blockToDelete.Hash)

		deletedHashMap[blockToDelete.Hash] = struct{}{}

		account.deleteAccountBlock(accounts, blockToDelete.Hash)
		account.rollbackLatestBlock()

		if blockToDelete.Height <= toBlock.Height {
			break
		}
	}

}
