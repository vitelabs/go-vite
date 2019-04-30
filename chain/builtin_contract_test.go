package chain

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestChain_builtInContract(t *testing.T) {
	chainInstance, accounts, snapshotBlockList := SetUp(17, 2654, 9)

	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testBuiltInContract(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("NewStorageDatabase", func(t *testing.T) {
		NewStorageDatabase(nil, chainInstance, accounts, snapshotBlockList)
	})

	t.Run("ConcurrentWrite", func(t *testing.T) {
		return
		var mu sync.RWMutex
		var wg sync.WaitGroup

		accounts := MakeAccounts(chainInstance, 5)

		wg.Add(2)
		testCount := 0
		const maxTestCount = 100
		go func() {
			defer wg.Done()
			for testCount < maxTestCount {
				// insert account blocks
				InsertAccountBlocks(&mu, chainInstance, accounts, rand.Intn(100))
				//snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, rand.Intn(1000), rand.Intn(20), false)...)

				// insert snapshot block
				snapshotBlock := createSnapshotBlock(chainInstance, false)

				mu.Lock()
				snapshotBlockList = append(snapshotBlockList, snapshotBlock)
				Snapshot(accounts, snapshotBlock)
				mu.Unlock()

				invalidBlocks, err := chainInstance.InsertSnapshotBlock(snapshotBlock)
				if err != nil {
					panic(err)
				}

				mu.Lock()
				DeleteInvalidBlocks(accounts, invalidBlocks)
				mu.Unlock()

				fmt.Printf("Snapshot Block: %d\n", snapshotBlock.Height)
				time.Sleep(100 * time.Millisecond)
				testCount++
			}

		}()

		go func() {
			defer wg.Done()

			for testCount < maxTestCount {
				mu.Lock()
				testSnapshotBlock := snapshotBlockList
				mu.Unlock()
				fmt.Printf("testSnapshotBlock: %d\n", len(testSnapshotBlock))
				NewStorageDatabase(&mu, chainInstance, accounts, testSnapshotBlock)
			}
		}()

		wg.Wait()
	})

	t.Run("GetRegisterList", func(t *testing.T) {
		//GetRegisterList(t, chainInstance)
	})

	return
}

func testBuiltInContractNoTesting(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {

	NewStorageDatabase(nil, chainInstance, accounts, snapshotBlockList)

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

func NewStorageDatabase(mu *sync.RWMutex, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	sbLen := len(snapshotBlockList)
	if sbLen <= 10 {
		return
	}

	//count := sbLen - 2

	for i := sbLen - 5; i >= 0 && i >= sbLen-10; i-- {
		//index := rand.Intn(count) + 2
		index := i
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

			// hackCheck
			current := uint64(0)
			for iter.Next() {
				next := chain_utils.BytesToUint64(iter.Key())

				if current+1 != next {
					panic("error")
				}
				current += 1

				keyStr := string(iter.Key())
				kv[keyStr] = make([]byte, len(iter.Value()))
				copy(kv[keyStr], iter.Value())
			}

			if err := iter.Error(); err != nil {
				panic(err)
			}

			if mu != nil {
				mu.RLock()
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

			if mu != nil {
				mu.RUnlock()
			}

			err = checkIterator(kv, func() (interfaces.StorageIterator, error) {
				return sd.NewStorageIterator(nil)
			})

			if err != nil {
				panic(fmt.Sprintf("account: %s, snapshotBlock: %+v.hashHeight: %+v. Error: %s", account.Addr, snapshotBlock,
					snapshotBlock.SnapshotContent[account.Addr], err.Error()))
			}

			for key, value := range kv {
				queryValue, err := sd.GetValue([]byte(key))
				if err != nil {
					panic("error")
				}
				if !bytes.Equal(value, queryValue) {
					panic(fmt.Sprintf("Addr: %s, snapshot height: %d, key: %d, value: %d, query value: %d",
						account.Addr, snapshotBlock.Height, []byte(key), value, queryValue))
				}

			}

		}
	}
}
