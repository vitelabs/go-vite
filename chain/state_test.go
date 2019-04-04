package chain

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func TestChain_State(t *testing.T) {

	chainInstance, accounts, snapshotBlockList := SetUp(t, 18, 910, 3)

	testState(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testState(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock) {
	t.Run("GetValue", func(t *testing.T) {
		GetValue(t, chainInstance, accounts)
	})

	t.Run("GetStorageIterator", func(t *testing.T) {
		GetStorageIterator(t, chainInstance, accounts)
	})

	t.Run("GetBalance", func(t *testing.T) {
		GetBalance(t, chainInstance, accounts)
	})

	t.Run("GetBalanceMap", func(t *testing.T) {
		GetBalanceMap(t, chainInstance, accounts)
	})
	t.Run("GetConfirmedBalanceList", func(t *testing.T) {
		GetConfirmedBalanceList(t, chainInstance, accounts, snapshotBlocks)
	})

	t.Run("GetContractMeta", func(t *testing.T) {
		GetContractMeta(t, chainInstance, accounts)
	})
	t.Run("GetContractCode", func(t *testing.T) {
		GetContractCode(t, chainInstance, accounts)
	})
	t.Run("GetContractList", func(t *testing.T) {
		GetContractList(t, chainInstance, accounts)
	})
	t.Run("GetVmLogList", func(t *testing.T) {
		GetVmLogList(t, chainInstance, accounts)
	})
	t.Run("GetQuotaUsed", func(t *testing.T) {
		GetQuotaUsed(t, chainInstance, accounts)
	})

}

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

func GetConfirmedBalanceList(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock) {

	for index, snapshotBlock := range snapshotBlocks {

		var addrList []types.Address
		balanceMap := make(map[types.Address]*big.Int)
		var highBlock *ledger.AccountBlock

		for _, account := range accounts {
			addrList = append(addrList, account.addr)
			highBlock = nil

			for i := index; i >= 0; i-- {
				confirmedBlockHashMap := account.ConfirmedBlockMap[snapshotBlocks[i].Hash]

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
					break
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
				t.Fatal(fmt.Sprintf("snapshotBlock %+v, content %+v, addr: %d, Balance: %d, Balance2: %d", snapshotBlock, snapshotBlock.SnapshotContent, addr.Bytes(), balance, balanceMap[addr]))
			}
		}
	}
}

func GetContractCode(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		code, err := chainInstance.GetContractCode(account.addr)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(code, account.Code) {
			t.Fatal("err")
		}
	}
}

func GetContractMeta(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		meta, err := chainInstance.GetContractMeta(account.addr)
		if err != nil {
			t.Fatal(err)
		}

		contractMeta := account.ContractMeta()
		if meta == nil {
			if contractMeta == nil {
				continue
			}
			t.Fatal("error")
		}
		if meta.Gid != contractMeta.Gid || meta.SendConfirmedTimes != contractMeta.SendConfirmedTimes {
			t.Fatal(fmt.Sprintf("%+v\n%+v\n", meta, contractMeta))
		}
	}
}

func GetContractList(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		contractMeta := account.ContractMeta()
		if contractMeta != nil {
			gid := contractMeta.Gid

			contractList, err := chainInstance.GetContractList(gid)
			if err != nil {
				t.Fatal(err)
			}

			if contractList[0] != account.addr {

				t.Fatal("error")
			}

		}
	}

	delegateContractList, err := chainInstance.GetContractList(types.DELEGATE_GID)
	if err != nil {
		t.Fatal(err)
	}
	if len(delegateContractList) < len(types.BuiltinContractAddrList) {
		t.Fatal("error")
	}
	for _, addr := range delegateContractList {
		if !types.IsBuiltinContractAddr(addr) {
			t.Fatal("error")
		}
	}
}

func GetVmLogList(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {

		for hash, vmLogList := range account.LogListMap {

			queryVmLogList, err := chainInstance.GetVmLogList(&hash)
			if err != nil {
				t.Fatal(err)
			}

			if len(queryVmLogList) != len(vmLogList) {
				t.Fatal(fmt.Sprintf("%+v\n%+v\n", vmLogList, queryVmLogList))
			}
			for index, vmLog := range queryVmLogList {

				if len(vmLog.Topics) != len(vmLogList[index].Topics) {
					t.Fatal("error")
				}
				for topicIndex, topic := range vmLog.Topics {
					if topic != vmLogList[index].Topics[topicIndex] {
						t.Fatal("error")
					}
				}

				if !bytes.Equal(vmLog.Data, vmLogList[index].Data) {
					t.Fatal("error")
				}
			}
		}

	}
}

func GetQuotaUsed(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	latestSb := chainInstance.GetLatestSnapshotBlock()
	sbList, err := chainInstance.GetSnapshotHeadersByHeight(latestSb.Height, false, 74)
	if err != nil {
		t.Fatal(err)
	}

	for _, account := range accounts {
		queryQuota, queryBlockCount := chainInstance.GetQuotaUsed(account.addr)

		var quota, blockCount uint64

		for hash := range account.unconfirmedBlocks {
			block := account.SendBlocksMap[hash]
			if block == nil {
				block = account.ReceiveBlocksMap[hash]
			}
			quota += block.AccountBlock.Quota
			blockCount += 1
		}
		for _, sb := range sbList {
			confirmedBlocks := account.ConfirmedBlockMap[sb.Hash]
			for hash := range confirmedBlocks {
				block := account.SendBlocksMap[hash]
				if block == nil {
					block = account.ReceiveBlocksMap[hash]
				}
				quota += block.AccountBlock.Quota
				blockCount += 1
			}
		}
		if queryQuota != quota || queryBlockCount != blockCount {
			t.Fatal(fmt.Sprintf("queryQuota: %d, quota: %d, queryBlockCount: %d, blockCount: %d",
				queryQuota, quota, queryBlockCount, blockCount))
		}

	}
}

func GetValue(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		keyValue := account.KeyValue()
		for key, value := range keyValue {
			queryValue, err := chainInstance.GetValue(account.addr, []byte(key))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(queryValue, value) {
				t.Fatal(fmt.Sprintf("queryValue: %+v\n value: %+v\n", queryValue, value))
			}
		}
	}
}

func GetStorageIterator(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		keyValue := account.KeyValue()
		err := checkIterator(keyValue, func() (interfaces.StorageIterator, error) {
			return chainInstance.GetStorageIterator(account.addr, nil)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func checkIterator(kvSet map[string][]byte, getIterator func() (interfaces.StorageIterator, error)) error {
	iter, err := getIterator()
	if err != nil {
		return err
	}
	count := 0
	for iter.Next() {
		count++
		key := iter.Key()

		value := iter.Value()
		if !bytes.Equal(kvSet[string(key)], value) {
			return errors.New(fmt.Sprintf("key: %d, kvValue:%d, value: %d", key, kvSet[string(key)], value))
		}
	}
	if err := iter.Error(); err != nil {
		return err
	}
	if count != len(kvSet) {
		return err
	}

	iterOk := iter.Last()
	count2 := 0
	for iterOk {
		count2++
		key := iter.Key()
		value := iter.Value()
		if !bytes.Equal(kvSet[string(key)], value) {
			fmt.Println(string(key))
			return errors.New(fmt.Sprintf("key: %s, kvValue:%d, value: %d", string(key), kvSet[string(key)], value))
		}
		iterOk = iter.Prev()
	}
	if err := iter.Error(); err != nil {
		return err
	}
	if count2 != len(kvSet) {
		return err
	}
	return nil

}
