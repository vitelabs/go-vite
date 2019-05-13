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

	chainInstance, accounts, snapshotBlockList := SetUp(2, 910, 3)

	testState(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testState(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock) {
	t.Run("GetValue", func(t *testing.T) {
		GetValue(chainInstance, accounts)
	})

	t.Run("GetStorageIterator", func(t *testing.T) {
		GetStorageIterator(chainInstance, accounts)
	})

	t.Run("GetBalance", func(t *testing.T) {
		GetBalance(chainInstance, accounts)
	})

	t.Run("GetBalanceMap", func(t *testing.T) {
		GetBalanceMap(chainInstance, accounts)
	})

	t.Run("GetConfirmedBalanceList", func(t *testing.T) {
		GetConfirmedBalanceList(chainInstance, accounts, snapshotBlocks)
	})

	t.Run("GetContractMeta", func(t *testing.T) {
		GetContractMeta(chainInstance, accounts)
	})

	t.Run("GetContractCode", func(t *testing.T) {
		GetContractCode(chainInstance, accounts)
	})

	t.Run("GetContractList", func(t *testing.T) {
		GetContractList(chainInstance, accounts)
	})

	t.Run("GetVmLogList", func(t *testing.T) {
		GetVmLogList(chainInstance, accounts)
	})

	t.Run("GetQuotaUsedList", func(t *testing.T) {
		GetQuotaUsed(chainInstance, accounts)
	})

}

func testStateNoTesting(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock) {
	GetValue(chainInstance, accounts)

	GetStorageIterator(chainInstance, accounts)

	GetBalance(chainInstance, accounts)

	GetBalanceMap(chainInstance, accounts)

	GetConfirmedBalanceList(chainInstance, accounts, snapshotBlocks)

	GetContractMeta(chainInstance, accounts)

	GetContractCode(chainInstance, accounts)

	GetContractList(chainInstance, accounts)

	GetVmLogList(chainInstance, accounts)

	GetQuotaUsed(chainInstance, accounts)
}

func GetBalance(chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		balance, err := chainInstance.GetBalance(addr, ledger.ViteTokenId)
		if err != nil {
			panic(err)
		}
		if balance == nil || balance.String() == "0" {
			if account.Balance().Cmp(account.GetInitBalance()) != 0 {
				panic(fmt.Sprintf("Error: %s, Balance %d, Balance2: %d, Balance3: %d", account.Addr, balance, account.Balance(), account.GetInitBalance()))
			}
		} else if balance.Cmp(account.Balance()) != 0 {
			panic(fmt.Sprintf("Error: %s, QueryBalance %d, Balance: %d", account.Addr, balance, account.Balance()))
		}

	}
}

func GetBalanceMap(chainInstance *chain, accounts map[types.Address]*Account) {
	for addr, account := range accounts {
		balanceMap, err := chainInstance.GetBalanceMap(addr)
		if err != nil {
			panic(err)
		}

		balance := balanceMap[ledger.ViteTokenId]

		if balance == nil || balance.String() == "0" {
			if account.Balance().Cmp(account.GetInitBalance()) != 0 {
				panic(fmt.Sprintf("Error: %s, Balance %d, Balance2: %d, Balance3: %d", account.Addr, balance, account.Balance(), account.GetInitBalance()))
			}
		} else if balanceMap[ledger.ViteTokenId].Cmp(account.Balance()) != 0 {
			panic(fmt.Sprintf("Error: Balance %d, balance2: %d", balanceMap[ledger.ViteTokenId], account.Balance()))
		}

	}
}

func GetConfirmedBalanceList(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlocks []*ledger.SnapshotBlock) {

	for index, snapshotBlock := range snapshotBlocks {

		var addrList []types.Address
		balanceMap := make(map[types.Address]*big.Int)
		var highBlock *ledger.AccountBlock

		for _, account := range accounts {
			addrList = append(addrList, account.Addr)
			highBlock = nil

			for i := index; i >= 0; i-- {
				confirmedBlockHashMap := account.ConfirmedBlockMap[snapshotBlocks[i].Hash]

				for hash := range confirmedBlockHashMap {
					block := account.BlocksMap[hash]

					if block == nil {
						panic(fmt.Sprintf("%s, %s", account.Addr, hash))
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
				balanceMap[account.Addr] = account.BalanceMap[highBlock.Hash]
			} else {
				balanceMap[account.Addr] = big.NewInt(0)
			}

		}
		queryBalanceMap, err := chainInstance.GetConfirmedBalanceList(addrList, ledger.ViteTokenId, snapshotBlock.Hash)
		if err != nil {
			panic(err)
		}
		for addr, balance := range queryBalanceMap {
			if balance.Cmp(balanceMap[addr]) != 0 {
				panic(fmt.Sprintf("snapshotBlock %+v, content %+v, addr: %s, highBlock: %+v, queryBalance: %d, Balance: %d", snapshotBlock, snapshotBlock.SnapshotContent, addr, highBlock, balance, balanceMap[addr]))
			}
		}
	}
}

func GetContractCode(chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		code, err := chainInstance.GetContractCode(account.Addr)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(code, account.Code) {
			panic("err")
		}
	}
}

func GetContractMeta(chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		meta, err := chainInstance.GetContractMeta(account.Addr)
		if err != nil {
			panic(err)
		}

		contractMeta := account.GetContractMeta()
		if meta == nil {
			if contractMeta == nil {
				continue
			}
			panic("error")
		} else {
			if contractMeta == nil {
				panic(fmt.Sprintf("%+v\n%+v\n", meta, contractMeta))
			}
		}
		if meta.Gid != contractMeta.Gid || meta.SendConfirmedTimes != contractMeta.SendConfirmedTimes {
			panic(fmt.Sprintf("%+v\n%+v\n", meta, contractMeta))
		}
	}
}

func GetContractList(chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		contractMeta := account.GetContractMeta()
		if contractMeta != nil {
			gid := contractMeta.Gid

			contractList, err := chainInstance.GetContractList(gid)
			if err != nil {
				panic(err)
			}
			if len(contractList) <= 0 {
				panic("error")
			}

			if contractList[0] != account.Addr {

				panic("error")
			}

		}
	}

	delegateContractList, err := chainInstance.GetContractList(types.DELEGATE_GID)
	if err != nil {
		panic(err)
	}
	if len(delegateContractList) < len(types.BuiltinContractAddrList) {
		panic("error")
	}
	for _, addr := range delegateContractList {
		if !types.IsBuiltinContractAddr(addr) {
			panic("error")
		}
	}
}

func GetVmLogList(chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {

		for hash, vmLogList := range account.LogListMap {

			queryVmLogList, err := chainInstance.GetVmLogList(&hash)
			if err != nil {
				panic(err)
			}

			if len(queryVmLogList) != len(vmLogList) {
				panic(fmt.Sprintf("%+v\n%+v\n", vmLogList, queryVmLogList))
			}
			for index, vmLog := range queryVmLogList {

				if len(vmLog.Topics) != len(vmLogList[index].Topics) {
					panic("error")
				}
				for topicIndex, topic := range vmLog.Topics {
					if topic != vmLogList[index].Topics[topicIndex] {
						panic("error")
					}
				}

				if !bytes.Equal(vmLog.Data, vmLogList[index].Data) {
					panic("error")
				}
			}
		}

	}
}

func GetQuotaUsed(chainInstance *chain, accounts map[types.Address]*Account) {
	latestSb := chainInstance.GetLatestSnapshotBlock()
	sbList, err := chainInstance.GetSnapshotHeadersByHeight(latestSb.Height, false, 74)
	if err != nil {
		panic(err)
	}

	for _, account := range accounts {
		quotaList := chainInstance.GetQuotaUsedList(account.Addr)
		var queryQuota, queryBlockCount uint64
		for _, quotaInfo := range quotaList {
			queryQuota += quotaInfo.QuotaTotal
			fmt.Println("queryQuota", queryQuota)
			queryBlockCount += quotaInfo.BlockCount
		}

		var quota, blockCount uint64

		for hash := range account.UnconfirmedBlocks {
			block := account.BlocksMap[hash]
			if block == nil {
				panic(fmt.Sprintf("error, hash: %s, UnconfirmedBlocks: %+v\n BlocksMap: %+v\n", hash, account.UnconfirmedBlocks, account.BlocksMap))
			}
			quota += block.Quota
			blockCount += 1
		}

		printIndex := 0

		for index := len(sbList) - 1; index >= 0; index-- {
			printIndex++
			sb := sbList[index]

			confirmedBlocks := account.ConfirmedBlockMap[sb.Hash]
			for hash := range confirmedBlocks {
				block := account.BlocksMap[hash]
				if block == nil {
					panic(fmt.Sprintf("error, UnconfirmedBlocks: %+v\n BlocksMap: %+v\n", account.UnconfirmedBlocks, account.BlocksMap))
				}
				quota += block.Quota
				blockCount += 1
			}
		}

		if queryQuota != quota || queryBlockCount != blockCount {
			panic(fmt.Sprintf("Addr: %s, queryQuota: %d, quota: %d, queryBlockCount: %d, blockCount: %d",
				account.Addr, queryQuota, quota, queryBlockCount, blockCount))
		}

	}
}

func GetValue(chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		keyValue := account.KeyValue()
		for key, value := range keyValue {
			queryValue, err := chainInstance.GetValue(account.Addr, []byte(key))
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(queryValue, value) {
				panic(fmt.Sprintf("queryValue: %+v\n value: %+v\n", queryValue, value))
			}
		}
	}
}

func GetStorageIterator(chainInstance *chain, accounts map[types.Address]*Account) {
	for _, account := range accounts {
		keyValue := account.KeyValue()
		err := checkIterator(keyValue, func() (interfaces.StorageIterator, error) {
			return chainInstance.GetStorageIterator(account.Addr, nil)
		})
		if err != nil {
			panic(fmt.Sprintf("%s, account: %s, account.latestAccountBlock: %+v\n", err.Error(), account.Addr, account.LatestBlock))
		}
	}
}

func checkIterator(kvSet map[string][]byte, getIterator func() (interfaces.StorageIterator, error)) error {
	iter, err := getIterator()
	if err != nil {
		return err
	}
	fmt.Printf("prepare check %d key\n", len(kvSet))
	count := 0
	for iter.Next() {
		count++
		if count > len(kvSet) {
			panic("too more key")
		}
		key := iter.Key()

		value := iter.Value()
		if !bytes.Equal(kvSet[string(key)], value) {
			kvSetStr := ""
			for key, value := range kvSet {
				kvSetStr += fmt.Sprintf("%d: %d, ", []byte(key), value)
			}
			return errors.New(fmt.Sprintf("key: %d, kv: %s, value: %d, queryValue: %d", key, kvSetStr, kvSet[string(key)], value))
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
		if count2 > len(kvSet) {
			panic(fmt.Sprintf("too more key, count2: %d, len(kvSet): %d", count2, len(kvSet)))
		}
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
