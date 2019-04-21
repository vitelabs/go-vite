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
	chainInstance, accounts, snapshotBlockList := SetUp(17, 2654, 9)

	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testBuiltInContract(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("NewStorageDatabase", func(t *testing.T) {
		NewStorageDatabase(chainInstance, accounts, snapshotBlockList)
	})
	t.Run("GetRegisterList", func(t *testing.T) {
		//GetRegisterList(t, chainInstance)
	})
}

func testBuiltInContractNoTesting(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {

	NewStorageDatabase(chainInstance, accounts, snapshotBlockList)

}

func TestVoteList(t *testing.T) {

	chainInstance, err := NewChainInstance("/Users/liyanda/test_ledger/ledger4/devdata", false)
	if err != nil {
		panic(err)
	}

	block := chainInstance.GetLatestSnapshotBlock()
	if err != nil {
		panic(err)
	}

	voteMap, err := chainInstance.GetVoteList(block.Hash, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}

	for _, voteInfo := range voteMap {
		addr := voteInfo.VoterAddr
		latest, e := chainInstance.GetLatestAccountBlock(addr)
		if e != nil {
			panic(e)
		}
		blocks, e := chainInstance.GetAccountBlocks(latest.Hash, 300)
		if e != nil {
			panic(e)
		}
		for i := 0; i < len(blocks); i++ {
			if blocks[i].ToAddress.String() == "vite_00000000000000000000000000000000000000042d7ef71894" {
				fmt.Printf("%s: %+v\n", blocks[i].AccountAddress, blocks[i].Data)
				break
			}
		}
	}
}

func GetRegisterList(t *testing.T, chainInstance *chain) {
	latestSnapshot := chainInstance.GetLatestSnapshotBlock()
	for i := uint64(1); i < latestSnapshot.Height; i++ {
		snapshotBlock, err := chainInstance.GetSnapshotHeaderByHeight(i)
		if err != nil {
			panic(err)
		}

		registerList, err := chainInstance.GetRegisterList(snapshotBlock.Hash, types.SNAPSHOT_GID)
		if err != nil {
			panic(err)
		}
		fmt.Println("")
		for _, register := range registerList {
			fmt.Println(register)
		}
		fmt.Println("")
	}
}

func NewStorageDatabase(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	sbLen := len(snapshotBlockList)
	if sbLen <= 2 {
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
				panic(err)
			}

			if *sd.Address() != account.Addr {
				panic("error")
			}

			prevSd, err := chainInstance.stateDB.NewStorageDatabase(prevSnapshotBlock.Hash, account.Addr)
			if err != nil {
				panic(err)
			}

			if *prevSd.Address() != account.Addr {
				panic("error")
			}

			kv := make(map[string][]byte)

			iter, err := prevSd.NewStorageIterator(nil)
			if err != nil {
				panic(err)
			}

			for iter.Next() {
				keyStr := string(iter.Key())
				kv[keyStr] = make([]byte, len(iter.Value()))
				copy(kv[keyStr], iter.Value())
			}

			if err := iter.Error(); err != nil {
				panic(err)
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
				panic(fmt.Sprintf("account: %s, snapshotBlock: %+v. Error: %s", account.Addr, snapshotBlock, err.Error()))
			}

			for key, value := range kv {
				queryValue, err := sd.GetValue([]byte(key))
				if err != nil {
					panic("error")
				}
				if !bytes.Equal(value, queryValue) {
					fmt.Printf("Addr: %s, snapshot height: %d, key: %s, kv: %+v, value: %d, query value: %d",
						account.Addr, snapshotBlock.Height, key, kv, value, queryValue)
					sd.GetValue([]byte(key))
					panic(fmt.Sprintf("Addr: %s, snapshot height: %d, key: %s, kv: %+v, value: %d, query value: %d",
						account.Addr, snapshotBlock.Height, key, kv, value, queryValue))
				}

			}

		}
	}
}
