package chain

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/rand"
	"testing"
)

func TestChain_builtInContract(t *testing.T) {
	chainInstance, accounts, snapshotBlockList := SetUp(t, 17, 2654, 9)

	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testBuiltInContract(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("NewStorageDatabase", func(t *testing.T) {
		NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)
	})
	t.Run("GetRegisterList", func(t *testing.T) {
		//GetRegisterList(t, chainInstance)
	})
}
func GetRegisterList(t *testing.T, chainInstance *chain) {
	latestSnapshot := chainInstance.GetLatestSnapshotBlock()
	for i := uint64(1); i < latestSnapshot.Height; i++ {
		snapshotBlock, err := chainInstance.GetSnapshotHeaderByHeight(i)
		if err != nil {
			t.Fatal(err)
		}

		registerList, err := chainInstance.GetRegisterList(snapshotBlock.Hash, types.SNAPSHOT_GID)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("")
		for _, register := range registerList {
			fmt.Println(register)
		}
		fmt.Println("")
	}
}

func NewStorageDatabase(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	sbLen := len(snapshotBlockList)
	if sbLen < 2 {
		return
	}

	count := sbLen - 2
	for i := 0; i < 10; i++ {
		index := rand.Intn(count) + 2
		snapshotBlock := snapshotBlockList[index]

		prevSnapshotBlock := snapshotBlockList[index-1]
		for _, account := range accounts {
			sd, err := chainInstance.stateDB.NewStorageDatabase(snapshotBlock.Hash, account.Addr)
			if err != nil {
				t.Fatal(err)
			}

			if *sd.Address() != account.Addr {
				t.Fatal("error")
			}

			prevSd, err := chainInstance.stateDB.NewStorageDatabase(prevSnapshotBlock.Hash, account.Addr)
			if err != nil {
				t.Fatal(err)
			}

			if *prevSd.Address() != account.Addr {
				t.Fatal("error")
			}

			kv := make(map[string][]byte)

			iter, err := prevSd.NewStorageIterator(nil)
			if err != nil {
				t.Fatal(err)
			}

			for iter.Next() {
				keyStr := string(iter.Key())
				kv[keyStr] = make([]byte, len(iter.Value()))
				copy(kv[keyStr], iter.Value())
			}

			if err := iter.Error(); err != nil {
				t.Fatal(err)
			}

			confirmedBlockHashMap := account.ConfirmedBlockMap[snapshotBlock.Hash]

			kvSetMap := make(map[types.Hash]map[string][]byte)
			for hash := range confirmedBlockHashMap {
				kvSetMap[hash] = account.KvSetMap[hash]
			}
			KvSetList := account.NewKvSetList(kvSetMap)
			for _, kvSet := range KvSetList {
				for k, v := range kvSet.Kv {
					kv[k] = v
				}

			}

			err = checkIterator(kv, func() (interfaces.StorageIterator, error) {
				return sd.NewStorageIterator(nil)
			})

			if err != nil {
				t.Fatal(fmt.Sprintf("account: %s, snapshotBlock: %+v. Error: %s", account.Addr, snapshotBlock, err.Error()))
			}

			for key, value := range kv {
				queryValue, err := sd.GetValue([]byte(key))
				if err != nil {
					t.Fatal("error")
				}
				if !bytes.Equal(value, queryValue) {
					t.Fatal(fmt.Sprintf("Addr: %s, snapshot block: %+v, key: %s, kv: %+v, value: %d, query value: %d",
						account.Addr, snapshotBlock, key, kv, value, queryValue))
				}

			}

		}
	}
}
