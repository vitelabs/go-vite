package chain

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func TestChain_AccountBlock(t *testing.T) {
	chainInstance, accounts, _ := SetUp(10, 5000, 90)

	testAccountBlock(t, chainInstance, accounts)
	TearDown(chainInstance)
}

func GetAccountBlockByHash(chainInstance Chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkAccountBlock(hash, func() (*ledger.AccountBlock, error) {
				return chainInstance.GetAccountBlockByHash(hash)
			})
			for _, sendBlock := range block.SendBlockList {
				checkAccountBlock(sendBlock.Hash, func() (*ledger.AccountBlock, error) {
					return chainInstance.GetAccountBlockByHash(sendBlock.Hash)
				})
			}

		}
	}

}

func GetAccountBlockByHeight(chainInstance Chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			checkAccountBlock(hash, func() (*ledger.AccountBlock, error) {
				return chainInstance.GetAccountBlockByHeight(account.Addr, block.Height)
			})
			for _, sendBlock := range block.SendBlockList {
				checkAccountBlock(hash, func() (*ledger.AccountBlock, error) {
					return chainInstance.GetAccountBlockByHeight(account.Addr, sendBlock.Height)
				})
			}
		}
	}
}

func IsAccountBlockExisted(chainInstance Chain, accounts map[types.Address]*Account) {

	for _, account := range accounts {
		for hash, block := range account.BlocksMap {
			ok, err := chainInstance.IsAccountBlockExisted(hash)
			if err != nil {
				panic(err)
			}
			if !ok {
				panic(fmt.Sprintf("error"))
			}
			for _, sendBlock := range block.SendBlockList {
				ok, err := chainInstance.IsAccountBlockExisted(sendBlock.Hash)
				if err != nil {
					panic(err)
				}
				if !ok {
					panic(fmt.Sprintf("error"))
				}
			}

		}
	}
}

func GetReceiveAbBySendAb(chainInstance Chain, accounts map[types.Address]*Account) {
	checkReceive := func(hash types.Hash) {
		receiveBlock, err := chainInstance.GetReceiveAbBySendAb(hash)
		if err != nil {
			panic(err)
		}
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			panic(err)
		}
		if block == nil {
			panic(fmt.Sprintf("hash is %s, block is %+v\n", hash, block))
		}

		if block.IsSendBlock() {
			receiveAccount := accounts[block.ToAddress]

			if receiveBlock != nil {
				if receiveBlock.AccountAddress != block.ToAddress {
					panic("error")
				}

				if receiveBlock.FromBlockHash != block.Hash {
					panic("error")
				}
				if _, ok := receiveAccount.ReceiveBlocksMap[receiveBlock.Hash]; !ok {
					panic("error")
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
					panic("error")
				}
			}
		} else {
			if receiveBlock != nil {
				panic("error")
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
func IsReceived(chainInstance *chain, accounts map[types.Address]*Account) {

	checkIsReceive := func(hash types.Hash) {
		received, err := chainInstance.IsReceived(hash)
		if err != nil {
			panic(err)
		}
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			panic(err)
		}
		if block == nil {
			panic(fmt.Sprintf("hash is %s", hash))
		}

		toAccount := accounts[block.ToAddress]
		if received {
			if block.IsReceiveBlock() {
				panic("error")
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
				panic(fmt.Sprintf("error. block is %+v, toAccount.ReceiveBlocksMap is %s\n", block, str))
			}
		} else {
			if block.IsReceiveBlock() {
				return
			}
			if _, ok := toAccount.ReceiveBlocksMap[hash]; ok {
				panic("error")
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

func GetAccountBlocks(chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		latestBlock := account.LatestBlock
		if latestBlock == nil {
			continue
		}
		for _, count := range []uint64{0, 10, 100, 1000, 10000} {

			blocks, err := chainInstance.GetAccountBlocks(latestBlock.Hash, count)
			if err != nil {
				panic(err)
			}
			checkBlocks(latestBlock, count, blocks)
			for _, block := range blocks {
				if block.AccountAddress != addr {
					panic("error")
				}
			}
		}
	}
}

func GetAccountBlocksByHeight(chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		latestBlock := account.LatestBlock
		if latestBlock == nil {
			continue
		}
		for _, count := range []uint64{0, 10, 100, 1000, 10000} {
			blocks, err := chainInstance.GetAccountBlocksByHeight(addr, latestBlock.Height, count)
			if err != nil {
				panic(err)
			}
			checkBlocks(latestBlock, count, blocks)
		}

	}
}

func checkBlocks(latestBlock *ledger.AccountBlock, count uint64, blocks []*ledger.AccountBlock) {
	blocksLength := uint64(len(blocks))
	if count > 0 && (blocksLength <= 0 || blocks[0].Height != latestBlock.Height) {
		panic(fmt.Sprintf("count is %d, LatestBlock is %+v, blocks is %+v\n", count, latestBlock, blocks))

	} else if count == 0 && blocksLength > 0 {
		panic("error")
	}

	if count < latestBlock.Height && blocksLength > latestBlock.Height {
		panic("error")

	} else if count > latestBlock.Height && blocksLength < latestBlock.Height {
		panic("error")

	}
	if blocksLength <= 0 {
		return
	}

	higherBlock := blocks[0]
	for i := uint64(1); i < blocksLength; i++ {
		currentBlock := blocks[i]
		if higherBlock.PrevHash != currentBlock.Hash || higherBlock.Height != currentBlock.Height+1 {
			panic("error")
		}

		higherBlock = currentBlock
	}
}
func GetConfirmedTimes(chainInstance Chain, accounts map[types.Address]*Account) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	checkConfirmedTimes := func(hash types.Hash) {
		times, err := chainInstance.GetConfirmedTimes(hash)
		if err != nil {
			panic(err)
		}

		firstConfirmSbHeight := latestSnapshotBlock.Height - times + 1

		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			panic(err)
		}

		if block == nil {
			panic(fmt.Sprintf("hash is %s, block is %+v\n", hash, block))
		}

		account := accounts[block.AccountAddress]
		if firstConfirmSbHeight > latestSnapshotBlock.Height {
			// no confirm
			if _, ok := account.UnconfirmedBlocks[hash]; !ok {
				panic(fmt.Sprintf("Addr: %s, hash is %s, UnconfirmedBlocks: %+v\n", block.AccountAddress, hash, account.UnconfirmedBlocks))
			}
		} else {
			firstConfirmSb, err := chainInstance.GetSnapshotBlockByHeight(firstConfirmSbHeight)
			if err != nil {
				panic(err)
			}

			blocksMap := account.ConfirmedBlockMap[firstConfirmSb.Hash]
			if _, ok := blocksMap[hash]; !ok {
				panic(fmt.Sprintf("error, %+v \n %+v \n", firstConfirmSb.SnapshotContent[block.AccountAddress], block))
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

func GetLatestAccountBlock(chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		block, err := chainInstance.GetLatestAccountBlock(addr)
		if err != nil {
			panic(err)
		}

		if block == nil {
			if account.LatestBlock != nil {
				fmt.Println(addr)
				chainInstance.GetLatestAccountBlock(addr)
				panic(fmt.Sprintf("%+v, %+v\n", block, account.LatestBlock))
			}
		} else if account.LatestBlock == nil {
			panic(fmt.Sprintf("%+v, %+v\n", block, account.LatestBlock))
		} else if block.Hash != account.LatestBlock.Hash {
			panic(fmt.Sprintf("%+v\n%+v\n", block, accounts[addr].LatestBlock))
		}

	}
}

func GetLatestAccountHeight(chainInstance Chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		height, err := chainInstance.GetLatestAccountHeight(addr)
		if err != nil {
			panic(err)
		}
		if height <= 0 {
			if account.LatestBlock == nil {
				continue
			} else {
				panic(fmt.Sprintf("%+v", account.LatestBlock))
			}
		} else if account.LatestBlock == nil {
			panic(fmt.Sprintf("%d, %+v", height, account.LatestBlock))
		}

		if height != account.LatestBlock.Height {
			panic(fmt.Sprintf("%d, %+v", height, account.LatestBlock))
		}
	}
}
func checkAccountBlock(hash types.Hash, getBlock func() (*ledger.AccountBlock, error)) {
	block, err := getBlock()

	if err != nil {
		panic(err)
	}

	if block == nil || block.Hash != hash {
		panic(fmt.Sprintf("hash is %s, block is error! block is %+v\n", hash, block))
	}
}

func testAccountBlockNoTesting(chainInstance *chain, accounts map[types.Address]*Account) {

	GetAccountBlockByHash(chainInstance, accounts)

	GetAccountBlockByHeight(chainInstance, accounts)

	IsAccountBlockExisted(chainInstance, accounts)

	IsReceived(chainInstance, accounts)

	GetReceiveAbBySendAb(chainInstance, accounts)

	GetConfirmedTimes(chainInstance, accounts)

	GetLatestAccountBlock(chainInstance, accounts)

	GetLatestAccountHeight(chainInstance, accounts)

	GetAccountBlocks(chainInstance, accounts)

	GetAccountBlocksByHeight(chainInstance, accounts)

}

func testAccountBlock(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {

	t.Run("GetAccountBlockByHash", func(t *testing.T) {
		GetAccountBlockByHash(chainInstance, accounts)
	})

	t.Run("GetAccountBlockByHeight", func(t *testing.T) {
		GetAccountBlockByHeight(chainInstance, accounts)
	})

	t.Run("IsAccountBlockExisted", func(t *testing.T) {
		IsAccountBlockExisted(chainInstance, accounts)
	})

	t.Run("IsReceived", func(t *testing.T) {
		IsReceived(chainInstance, accounts)
	})

	t.Run("GetReceiveAbBySendAb", func(t *testing.T) {
		GetReceiveAbBySendAb(chainInstance, accounts)
	})

	t.Run("GetConfirmedTimes", func(t *testing.T) {
		GetConfirmedTimes(chainInstance, accounts)
	})

	t.Run("GetLatestAccountBlock", func(t *testing.T) {
		GetLatestAccountBlock(chainInstance, accounts)
	})

	t.Run("GetLatestAccountHeight", func(t *testing.T) {
		GetLatestAccountHeight(chainInstance, accounts)
	})

	t.Run("GetAccountBlocks", func(t *testing.T) {
		GetAccountBlocks(chainInstance, accounts)
	})

	t.Run("GetAccountBlocksByHeight", func(t *testing.T) {
		GetAccountBlocksByHeight(chainInstance, accounts)
	})
}
