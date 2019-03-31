package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func GetBalance(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		balance, err := chainInstance.GetBalance(addr, ledger.ViteTokenId)
		if err != nil {
			t.Fatal(err)
		}

		if balance.Cmp(account.BalanceMap[account.latestBlock.Hash]) != 0 {
			t.Fatal(fmt.Sprintf("Error: balance %d, balance2: %d", balance, account.BalanceMap[account.latestBlock.Hash]))
		}

	}
}

func GetBalanceMap(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		balanceMap, err := chainInstance.GetBalanceMap(addr)
		receiveBlocksLen := len(account.ReceiveBlocksMap)
		if err != nil {
			t.Fatal(err)
		}
		if len(balanceMap) <= 0 && receiveBlocksLen > 0 {
			t.Fatal("error")
		}

		if len(balanceMap) > 0 {
			if balanceMap[ledger.ViteTokenId].Cmp(account.BalanceMap[account.latestBlock.Hash]) != 0 {
				t.Fatal(fmt.Sprintf("Error: balance %d, balance2: %d", balanceMap[ledger.ViteTokenId], account.BalanceMap[account.latestBlock.Hash]))
			}
		}
	}
}

func GetConfirmedBalanceList(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {

	latestSb := chainInstance.GetLatestSnapshotBlock()
	snapshotBlocks, err := chainInstance.GetSnapshotBlocks(latestSb.Hash, false, 20)
	if err != nil {
		t.Fatal(err)
	}

	for _, snapshotBlock := range snapshotBlocks {

		var addrList []types.Address
		balanceMap := make(map[types.Address]*big.Int)

		for _, account := range accounts {
			addrList = append(addrList, account.addr)

			confirmedBlockHashMap := account.ConfirmedBlockMap[snapshotBlock.Hash]
			var highBlock *ledger.AccountBlock

			for hash := range confirmedBlockHashMap {
				block, err := chainInstance.GetAccountBlockByHash(hash)
				if err != nil {
					t.Fatal(err)
				}

				if highBlock == nil || block.Height > highBlock.Height {
					highBlock = block
				}
			}
			if highBlock != nil {
				balanceMap[account.addr] = account.BalanceMap[highBlock.Hash]
			} else {
				balanceMap[account.addr] = big.NewInt(0)
			}
		}
		queryBalanceMap, err := chainInstance.GetConfirmedBalanceList(addrList, ledger.ViteTokenId, snapshotBlock.Hash)
		if err != nil {
			t.Fatal(err)
		}
		for addr, balance := range queryBalanceMap {
			if balance.Cmp(balanceMap[addr]) != 0 {
				t.Fatal("error")
			} else {
				fmt.Println(balance)
			}
		}
	}
}
