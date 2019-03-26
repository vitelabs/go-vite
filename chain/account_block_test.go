package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

func GetAccountBlockByHash(t *testing.T, chainInstance Chain, hashList []types.Hash) {
	for _, hash := range hashList {
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if block == nil || block.Hash != hash {
			t.Fatal(fmt.Sprintf("block is error! block is %+v\n", block))
		}
	}
}

func GetAccountBlockByHeight(t *testing.T, chainInstance Chain, addrList []types.Address, heightList []uint64) {
	for index, height := range heightList {
		block, err := chainInstance.GetAccountBlockByHeight(addrList[index], height)
		if err != nil {
			t.Fatal(err)
		}
		if block.AccountAddress != addrList[index] ||
			block.Height != height {
			t.Fatal("block is error!")
		}
	}
}

func IsAccountBlockExisted(t *testing.T, chainInstance Chain, hashList []types.Hash) {
	for _, hash := range hashList {
		ok, err := chainInstance.IsAccountBlockExisted(hash)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal(fmt.Sprintf("error"))
		}
	}
}

func GetReceiveAbBySendAb(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, hashList []types.Hash) {
	for _, hash := range hashList {
		receiveBlock, err := chainInstance.GetReceiveAbBySendAb(hash)
		if err != nil {
			t.Fatal(err)
		}
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
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
					if receiveBlock.AccountBlock.FromBlockHash == hash {
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
}
func IsReceived(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, hashList []types.Hash) {
	//sendBlockHash types.Hash
	for _, hash := range hashList {
		received, err := chainInstance.IsReceived(hash)
		if err != nil {
			t.Fatal(err)
		}
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}

		toAccount := accounts[block.ToAddress]
		if received {
			if block.IsReceiveBlock() {
				t.Fatal("error")
			}
			hasReceived := false
			for _, receiveBlock := range toAccount.ReceiveBlocksMap {
				if receiveBlock.AccountBlock.FromBlockHash == hash {
					hasReceived = true
					break
				}
			}
			if !hasReceived {
				t.Fatal(fmt.Sprintf("error. block is %+v", block))
			}
		} else {
			if block.IsReceiveBlock() {
				continue
			}
			if _, ok := toAccount.ReceiveBlocksMap[hash]; ok {
				t.Fatal("error")
			}
		}
	}
}

func GetAccountBlocks(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, addrList []types.Address) {
	for _, addr := range addrList {
		latestBlock := accounts[addr].latestBlock
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

func GetAccountBlocksByHeight(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, addrList []types.Address) {
	for _, addr := range addrList {
		latestBlock := accounts[addr].latestBlock
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
	if count != 0 && blocks[0].Height != latestBlock.Height {
		t.Fatal("error")
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
func GetConfirmedTimes(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, hashList []types.Hash) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	for index, hash := range hashList {
		times, err := chainInstance.GetConfirmedTimes(hash)
		if err != nil {
			t.Fatal(err)
		}

		firstConfirmSbHeight := latestSnapshotBlock.Height - times + 1

		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}

		account := accounts[block.AccountAddress]
		if firstConfirmSbHeight > latestSnapshotBlock.Height {
			for _, blocksMap := range account.ConfirmedBlockMap {
				if _, ok := blocksMap[hash]; ok {
					t.Fatal("error")
				}
			}
		} else {
			firstConfirmSb, err := chainInstance.GetSnapshotBlockByHeight(firstConfirmSbHeight)
			if err != nil {
				t.Fatal(err)
			}

			blocksMap := account.ConfirmedBlockMap[firstConfirmSb.Hash]
			if _, ok := blocksMap[hash]; !ok {
				t.Fatal(fmt.Printf("error, %+v\n%+v", firstConfirmSb.SnapshotContent[block.AccountAddress], block))
			}
		}

	}
}

func GetLatestAccountBlock(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, addrList []types.Address) {
	for _, addr := range addrList {
		block, err := chainInstance.GetLatestAccountBlock(addr)
		if err != nil {
			t.Fatal(err)
		}

		if block.Hash != accounts[addr].latestBlock.Hash {
			t.Fatal(fmt.Sprintf("%+v\n%+v\n", block, accounts[addr].latestBlock))
			chainInstance.GetLatestAccountBlock(addr)
		}
	}
}

func GetLatestAccountHeight(t *testing.T, chainInstance Chain, accounts map[types.Address]*Account, addrList []types.Address) {
	for _, addr := range addrList {
		height, err := chainInstance.GetLatestAccountHeight(addr)
		if err != nil {
			t.Fatal(err)
		}

		if height != accounts[addr].latestBlock.Height {
			t.Fatal("error")
		}
	}
}

//func GetAccountBlocks(t *testing.T, chainInstance Chain, map[types.Address]*Account, addrList []types.Address)  {
//	for _, addr := range addrList {
//		blocks :=
//	}
//}
