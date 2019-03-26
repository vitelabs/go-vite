package chain

import (
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
		if balance.Cmp(big.NewInt(int64(len(account.ReceiveBlocksMap)*100))) != 0 {
			t.Fatal("error")
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
			if receiveBlocksLen <= 0 {
				t.Fatal("error")
			}
			if balanceMap[ledger.ViteTokenId].Cmp(big.NewInt(int64(receiveBlocksLen*100))) != 0 {
				t.Fatal("error")
			}
		}
	}
}
