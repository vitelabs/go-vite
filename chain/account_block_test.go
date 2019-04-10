package chain

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func TestChain_AccountBlock(t *testing.T) {
	chainInstance, accounts, _ := SetUp(t, 10, 5000, 90)

	testAccountBlock(t, chainInstance, accounts)
	TearDown(chainInstance)
}

func GetAccountBlockByHash(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkAccountBlock(t, hash, func() (*ledger.AccountBlock, error) {
				return chainInstance.GetAccountBlockByHash(hash)
			})
			for _, sendBlock := range block.SendBlockList {
				checkAccountBlock(t, sendBlock.Hash, func() (*ledger.AccountBlock, error) {
					return chainInstance.GetAccountBlockByHash(sendBlock.Hash)
				})
			}

		}
	}

}

func GetAccountBlockByHeight(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkAccountBlock(t, hash, func() (*ledger.AccountBlock, error) {
				return chainInstance.GetAccountBlockByHeight(account.Addr, block.Height)
			})
			for _, sendBlock := range block.SendBlockList {
				checkAccountBlock(t, hash, func() (*ledger.AccountBlock, error) {
					return chainInstance.GetAccountBlockByHeight(account.Addr, sendBlock.Height)
				})
			}
		}
	}
}

func IsAccountBlockExisted(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {

	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			ok, err := chainInstance.IsAccountBlockExisted(hash)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal(fmt.Sprintf("error"))
			}
			for _, sendBlock := range block.SendBlockList {
				ok, err := chainInstance.IsAccountBlockExisted(sendBlock.Hash)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal(fmt.Sprintf("error"))
				}
			}

		}
	}
}

func GetReceiveAbBySendAb(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {
	checkReceive := func(hash types.Hash) {
		receiveBlock, err := chainInstance.GetReceiveAbBySendAb(hash)
		if err != nil {
			t.Fatal(err)
		}
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if block == nil {
			t.Fatal(fmt.Sprintf("hash is %s, block is %+v\n", hash, block))
		}

		if block.IsSendBlock() {
			receiveAccount := accounts[block.ToAddress]

			if receiveBlock != nil {
				if receiveBlock.AccountAddress != block.ToAddress {
					t.Fatal("error")
				}

				if receiveBlock.FromBlockHash != block.Hash {
					t.Fatal("error")
				}
				if _, ok := receiveAccount.ReceiveBlocksMap[receiveBlock.Hash]; !ok {
					t.Fatal("error")
				}
			} else {
				hasReceived := false
				for _, receiveBlock := range receiveAccount.ReceiveBlocksMap {
					if receiveBlock.FromBlockHash == hash {
						hasReceived = true
						break
					}
				}
				if hasReceived {
					t.Fatal("error")
				}
			}
		} else {
			if receiveBlock != nil {
				t.Fatal("error")
			}
		}
	}

	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkReceive(hash)
			for _, sendBlock := range block.SendBlockList {
				checkReceive(sendBlock.Hash)
			}
		}
	}
}
func IsReceived(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {

	checkIsReceive := func(hash types.Hash) {
		received, err := chainInstance.IsReceived(hash)
		if err != nil {
			t.Fatal(err)
		}
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if block == nil {
			t.Fatal(fmt.Sprintf("hash is %s", hash))
		}

		toAccount := accounts[block.ToAddress]
		if received {
			if block.IsReceiveBlock() {
				t.Fatal("error")
			}
			hasReceived := false
			for _, receiveBlock := range toAccount.ReceiveBlocksMap {
				if receiveBlock.FromBlockHash == hash {
					hasReceived = true
					break
				}
			}
			if !hasReceived {
				str := ""
				for _, block := range toAccount.ReceiveBlocksMap {
					str += fmt.Sprintf("%+v\n", block)
				}
				t.Fatal(fmt.Sprintf("error. block is %+v, toAccount.ReceiveBlocksMap is %s\n", block, str))
			}
		} else {
			if block.IsReceiveBlock() {
				return
			}
			if _, ok := toAccount.ReceiveBlocksMap[hash]; ok {
				t.Fatal("error")
			}
		}
	}
	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkIsReceive(hash)
			for _, sendBlock := range block.SendBlockList {
				checkIsReceive(sendBlock.Hash)
			}
		}
	}

}

func GetAccountBlocks(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		latestBlock := account.LatestBlock
		if latestBlock == nil {
			continue
		}
		for _, count := range []uint64{0, 10, 100, 1000, 10000} {

			blocks, err := chainInstance.GetAccountBlocks(latestBlock.Hash, count)
			if err != nil {
				t.Fatal(err)
			}
			checkBlocks(t, latestBlock, count, blocks)
			for _, block := range blocks {
				if block.AccountAddress != addr {
					t.Fatal("error")
				}
			}
		}
	}
}

func GetAccountBlocksByHeight(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		latestBlock := account.LatestBlock
		if latestBlock == nil {
			continue
		}
		for _, count := range []uint64{0, 10, 100, 1000, 10000} {
			blocks, err := chainInstance.GetAccountBlocksByHeight(addr, latestBlock.Height, count)
			if err != nil {
				t.Fatal(err)
			}
			checkBlocks(t, latestBlock, count, blocks)
		}

	}
}

func checkBlocks(t *testing.T, latestBlock *ledger.AccountBlock, count uint64, blocks []*ledger.AccountBlock) {
	blocksLength := uint64(len(blocks))
	if count > 0 && (blocksLength <= 0 || blocks[0].Height != latestBlock.Height) {
		t.Fatal(fmt.Sprintf("count is %d, LatestBlock is %+v, blocks is %+v\n", count, latestBlock, blocks))

	} else if count == 0 && blocksLength > 0 {
		t.Fatal("error")
	}

	if count < latestBlock.Height && blocksLength > latestBlock.Height {
		t.Fatal("error")

	} else if count > latestBlock.Height && blocksLength < latestBlock.Height {
		t.Fatal("error")

	}
	if blocksLength <= 0 {
		return
	}

	higherBlock := blocks[0]
	for i := uint64(1); i < blocksLength; i++ {
		currentBlock := blocks[i]
		if higherBlock.PrevHash != currentBlock.Hash || higherBlock.Height != currentBlock.Height+1 {
			t.Fatal("error")
		}

		higherBlock = currentBlock
	}
}
func GetConfirmedTimes(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	checkConfirmedTimes := func(hash types.Hash) {
		times, err := chainInstance.GetConfirmedTimes(hash)
		if err != nil {
			t.Fatal(err)
		}

		firstConfirmSbHeight := latestSnapshotBlock.Height - times + 1

		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}

		if block == nil {
			t.Fatal(fmt.Sprintf("hash is %s, block is %+v\n", hash, block))
		}

		account := accounts[block.AccountAddress]
		if firstConfirmSbHeight > latestSnapshotBlock.Height {
			// no confirm
			if _, ok := account.UnconfirmedBlocks[hash]; !ok {
				t.Fatal(fmt.Sprintf("Addr: %s, hash is %s, UnconfirmedBlocks: %+v\n", block.AccountAddress, hash, account.UnconfirmedBlocks))
			}
		} else {
			firstConfirmSb, err := chainInstance.GetSnapshotBlockByHeight(firstConfirmSbHeight)
			if err != nil {
				t.Fatal(err)
			}

			blocksMap := account.ConfirmedBlockMap[firstConfirmSb.Hash]
			if _, ok := blocksMap[hash]; !ok {
				t.Fatal(fmt.Sprintf("error, %+v \n %+v \n", firstConfirmSb.SnapshotContent[block.AccountAddress], block))
			}
		}
	}
	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkConfirmedTimes(hash)
			for _, sendBlock := range block.SendBlockList {
				checkConfirmedTimes(sendBlock.Hash)
			}
		}
	}
}

func GetLatestAccountBlock(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		block, err := chainInstance.GetLatestAccountBlock(addr)
		if err != nil {
			t.Fatal(err)
		}

		if block == nil {
			if account.LatestBlock != nil {
				chainInstance.GetLatestAccountBlock(addr)
				t.Fatal(fmt.Sprintf("%+v, %+v\n", block, account.LatestBlock))
			}
		} else if account.LatestBlock == nil {
			t.Fatal(fmt.Sprintf("%+v, %+v\n", block, account.LatestBlock))
		} else if block.Hash != account.LatestBlock.Hash {
			t.Fatal(fmt.Sprintf("%+v\n%+v\n", block, accounts[addr].LatestBlock))
		}

	}
}

func GetLatestAccountHeight(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		height, err := chainInstance.GetLatestAccountHeight(addr)
		if err != nil {
			t.Fatal(err)
		}
		if height <= 0 {
			if account.LatestBlock == nil {
				continue
			} else {
				t.Fatal(fmt.Sprintf("%+v", account.LatestBlock))
			}
		} else if account.LatestBlock == nil {
			t.Fatal(fmt.Sprintf("%d, %+v", height, account.LatestBlock))
		}

		if height != account.LatestBlock.Height {
			t.Fatal(fmt.Sprintf("%d, %+v", height, account.LatestBlock))
		}
	}
}
func checkAccountBlock(t *testing.T, hash types.Hash, getBlock func() (*ledger.AccountBlock, error)) {
	block, err := getBlock()

	if err != nil {
		t.Fatal(err)
	}

	if block == nil || block.Hash != hash {
		t.Fatal(fmt.Sprintf("hash is %s, block is error! block is %+v\n", hash, block))
	}
}

func testAccountBlock(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {

	t.Run("GetAccountBlockByHash", func(t *testing.T) {
		GetAccountBlockByHash(t, chainInstance, accounts)
	})

	t.Run("GetAccountBlockByHeight", func(t *testing.T) {
		GetAccountBlockByHeight(t, chainInstance, accounts)
	})

	t.Run("IsAccountBlockExisted", func(t *testing.T) {
		IsAccountBlockExisted(t, chainInstance, accounts)
	})

	t.Run("IsReceived", func(t *testing.T) {
		IsReceived(t, chainInstance, accounts)
	})

	t.Run("GetReceiveAbBySendAb", func(t *testing.T) {
		GetReceiveAbBySendAb(t, chainInstance, accounts)
	})

	t.Run("GetConfirmedTimes", func(t *testing.T) {
		GetConfirmedTimes(t, chainInstance, accounts)
	})

	t.Run("GetLatestAccountBlock", func(t *testing.T) {
		GetLatestAccountBlock(t, chainInstance, accounts)
	})

	t.Run("GetLatestAccountHeight", func(t *testing.T) {
		GetLatestAccountHeight(t, chainInstance, accounts)
	})

	t.Run("GetAccountBlocks", func(t *testing.T) {
		GetAccountBlocks(t, chainInstance, accounts)
	})

	t.Run("GetAccountBlocksByHeight", func(t *testing.T) {
		GetAccountBlocksByHeight(t, chainInstance, accounts)
	})
}
