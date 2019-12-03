package chain

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/common/fork"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"testing"
	"time"

	"encoding/gob"
	"fmt"
	"github.com/docker/docker/pkg/reexec"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/quota"
	"math/rand"
	"os/exec"
	"sync"
	"syscall"
)

var GenesisJson = `{
  "GenesisAccountAddress": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
  "ForkPoints": {
  },
  "GovernanceInfo": {
    "ConsensusGroupInfoMap":{
      "00000000000000000001":{
        "NodeCount": 1,
        "Interval":1,
        "PerCount":3,
        "RandCount":2,
        "RandRank":100,
        "Repeat":1,
        "CheckLevel":0,
        "CountingTokenId":"tti_5649544520544f4b454e6e40",
        "RegisterConditionId":1,
        "RegisterConditionParam":{
          "StakeAmount": 100000000000000000000000,
          "StakeHeight": 1,
          "StakeToken": "tti_5649544520544f4b454e6e40"
        },
        "VoteConditionId":1,
        "VoteConditionParam":{},
        "Owner":"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
        "StakeAmount":0,
        "ExpirationHeight":1
      },
      "00000000000000000002":{
        "NodeCount": 1,
        "Interval":3,
        "PerCount":1,
        "RandCount":2,
        "RandRank":100,
        "Repeat":48,
        "CheckLevel":1,
        "CountingTokenId":"tti_5649544520544f4b454e6e40",
        "RegisterConditionId":1,
        "RegisterConditionParam":{
          "StakeAmount": 100000000000000000000000,
          "StakeHeight": 1,
          "StakeToken": "tti_5649544520544f4b454e6e40"
        },
        "VoteConditionId":1,
        "VoteConditionParam":{},
        "Owner":"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
        "StakeAmount":0,
        "ExpirationHeight":1
      }
    },

    "RegistrationInfoMap":{
      "00000000000000000001":{
        "s1":{
          "BlockProducingAddress":"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a",
          "StakeAddress":"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a",
          "Amount":100000000000000000000000,
          "ExpirationHeight":7776000,
          "RewardTime":1,
          "RevokeTime":0,
          "HistoryAddressList":["vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a"]
        }
      }
    }
  },
  "AssetInfo":{
    "TokenInfoMap":{
      "tti_5649544520544f4b454e6e40":{
        "TokenName":"Vite Token",
        "TokenSymbol":"VITE",
        "TotalSupply":1000000000000000000000000000,
        "Decimals":18,
        "Owner":"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
        "MaxSupply":115792089237316195423570985008687907853269984665640564039457584007913129639935,
        "IsOwnerBurnOnly":false,
        "IsReIssuable":true
      }
    },
    "LogList": [
      {
        "Data": "",
        "Topics": [
          "3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6",
          "000000000000000000000000000000000000000000005649544520544f4b454e"
        ]
      }
    ]
  },
  "QuotaInfo": {
    "StakeInfoMap": {
      "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a": [
        {
          "Amount": 1000000000000000000000,
          "ExpirationHeight": 259200,
          "Beneficiary": "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a"
        },
        {
          "Amount": 1000000000000000000000,
          "ExpirationHeight": 259200,
          "Beneficiary": "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25"
        },
        {
          "Amount": 1000000000000000000000,
          "ExpirationHeight": 259200,
          "Beneficiary": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"
        },
        {
          "Amount": 1000000000000000000000,
          "ExpirationHeight": 259200,
          "Beneficiary": "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23"
        }
      ]
    },
    "StakeBeneficialMap":{
      "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a":1000000000000000000000,
      "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25":1000000000000000000000,
      "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a":1000000000000000000000,
      "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23":1000000000000000000000
    }
  },
  "AccountBalanceMap": {
    "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a": {
      "tti_5649544520544f4b454e6e40":99996000000000000000000000
    },
    "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": {
      "tti_5649544520544f4b454e6e40":100000000000000000000000000
    },
    "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a": {
      "tti_5649544520544f4b454e6e40":600000000000000000000000000
    },
    "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25": {
      "tti_5649544520544f4b454e6e40":100000000000000000000000000
    },
    "vite_847e1672c9a775ca0f3c3a2d3bf389ca466e5501cbecdb7107": {
      "tti_5649544520544f4b454e6e40":100000000000000000000000000
    }
  }
}
`

func NewChainInstance(dirName string, clear bool) (*chain, error) {
	var dataDir string

	if path.IsAbs(dirName) {
		dataDir = dirName
	} else {
		dataDir = path.Join(test_tools.DefaultDataDir(), dirName)
	}

	if clear {
		os.RemoveAll(dataDir)
	}
	genesisConfig := &config.Genesis{}

	json.Unmarshal([]byte(GenesisJson), genesisConfig)

	chainInstance := NewChain(dataDir, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		return nil, err
	}
	// mock consensus
	chainInstance.SetConsensus(&test_tools.MockConsensus{})

	chainInstance.Start()
	return chainInstance, nil
}

func Clear(c *chain) error {
	return os.RemoveAll(c.dataDir)
}

func SetUp(accountNum, txCount, snapshotPerBlockNum int) (*chain, map[types.Address]*Account, []*ledger.SnapshotBlock) {
	// set fork point

	if len(fork.GetActiveForkPointList()) <= 0 {
		fork.SetForkPoints(&config.ForkPoints{
			SeedFork: &config.ForkPoint{
				Version: 1,
				Height:  10000000,
			},
		})
	}

	// test quota
	quota.InitQuotaConfig(true, true)

	chainInstance, err := NewChainInstance("unit_test/devdata", false)
	if err != nil {
		panic(err)
	}

	chainInstance.ResetLog(chainInstance.chainDir, "info")
	//InsertSnapshotBlock(chainInstance, true)

	accounts := MakeAccounts(chainInstance, accountNum)

	snapshotBlockList := InsertAccountBlockAndSnapshot(chainInstance, accounts, txCount, snapshotPerBlockNum, false)

	return chainInstance, accounts, snapshotBlockList
}

func TearDown(chainInstance *chain) {
	chainInstance.Stop()
	chainInstance.Destroy()
}

func TestChain(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// test panic
	//tempChain, accounts, snapshotBlockList := SetUp(20, 1, 3)
	//TearDown(tempChain)
	//testPanic(t, accounts, snapshotBlockList)

	// test insert
	chainInstance, accounts, snapshotBlockList := SetUp(20, 500, 10)

	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	// test insert and query
	snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, rand.Intn(300), rand.Intn(5), true)...)

	// test all
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	// test insert & delete
	snapshotBlockList = testInsertAndDelete(t, chainInstance, accounts, snapshotBlockList)

	// test panic
	TearDown(chainInstance)
}

func testChainAll(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	// account
	testAccount(chainInstance, accounts)

	// account block
	testAccountBlock(t, chainInstance, accounts)

	// on road
	testOnRoad(t, chainInstance, accounts)

	// snapshot block
	testSnapshotBlock(t, chainInstance, accounts, snapshotBlockList)

	// state
	testState(t, chainInstance, accounts, snapshotBlockList)

	// built-in contract
	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)
}

func testChainAllNoTesting(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	// account
	testAccount(chainInstance, accounts)

	// unconfirmed
	testUnconfirmedNoTesting(chainInstance, accounts)

	// account block
	testAccountBlockNoTesting(chainInstance, accounts)

	// on road
	testOnRoadNoTesting(chainInstance, accounts)

	// snapshot block
	testSnapshotBlockNoTesting(chainInstance, accounts, snapshotBlockList)

	// state
	testStateNoTesting(chainInstance, accounts, snapshotBlockList)

	// built-in contract
	testBuiltInContractNoTesting(chainInstance, accounts, snapshotBlockList)
}

func TestCheckHash(t *testing.T) {
	chainInstance, _, _ := SetUp(0, 0, 0)
	if err := chainInstance.CheckHash(); err != nil {
		panic(err)
	}
}

func TestCheckHash2(t *testing.T) {
	chainInstance, _, _ := SetUp(0, 0, 0)
	hash, err := types.HexToHash("3cc090aaaa241b3ff480cd461a1fb220fd429717855b5c990d1cb34dd1cef6c1")
	if err != nil {
		t.Fatal(err)
	}

	block, err := chainInstance.GetAccountBlockByHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)
}

func testPanic(t *testing.T, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {

	//for i := 0; i < 10; i++ {
	saveData(accounts, snapshotBlockList)
	accounts = nil
	snapshotBlockList = nil

	for j := 0; j < 5; j++ {
		cmd := reexec.Command("randomPanic")

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		t.Run(fmt.Sprintf("panic %d_%d", 1, j+1), func(t *testing.T) {
			err := cmd.Run()

			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					if status.ExitStatus() == mockPanicExitStatus {
						return
					}
				}
			}
			//panic(fmt.Sprintf("cmd.Run(): %v", err))
			fmt.Printf("cmd.Run(): %v", err)

		})

	}

	//}

}

func init() {
	reexec.Register("randomPanic", randomPanic)
	if reexec.Init() {
		os.Exit(0)
	}
}

const mockPanicExitStatus = 16

func randomPanic() {
	quota.InitQuotaConfig(true, true)
	chainInstance, err := NewChainInstance("unit_test", false)

	accounts, snapshotBlockList := loadData(chainInstance)
	if len(accounts) <= 0 {
		accounts = MakeAccounts(chainInstance, 100)
		snapshotBlockList = InsertAccountBlockAndSnapshot(chainInstance, accounts, 100, 10, false)
		if err != nil {
			panic(err)
		}
	}

	snapshotBlockList = recoverAfterPanic(chainInstance, accounts, snapshotBlockList)

	testChainAllNoTesting(chainInstance, accounts, snapshotBlockList)

	if err != nil {
		panic(err)
	}

	var mu sync.RWMutex

	defer func() {
		mu.Lock()
		saveData(accounts, snapshotBlockList)
		mu.Unlock()

		//os.Exit(mockPanicExitStatus)
	}()
	//go func() {
	//	defer func() {
	//		mu.Lock()
	//		saveData(accounts, snapshotBlockList)
	//		mu.Unlock()
	//
	//		os.Exit(mockPanicExitStatus)
	//	}()
	//
	//	fmt.Println("Wait 2 seconds")
	//	time.Sleep(time.Second * 2)
	//
	//	for {
	//		random := rand.Intn(100)
	//		if random > 90 {
	//			panic("error")
	//		}
	//		time.Sleep(time.Microsecond * 10)
	//	}
	//
	//}()

	for {
		// insert account blocks
		InsertAccountBlocks(&mu, chainInstance, accounts, rand.Intn(1000))
		//snapshotBlockList = append(snapshotBlockList, InsertAccountBlockAndSnapshot(chainInstance, accounts, rand.Intn(1000), rand.Intn(20), false)...)

		// insert snapshot block
		snapshotBlock := createSnapshotBlock(chainInstance, createSbOption{
			SnapshotAll: false,
		})

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

	}

}

func recoverAfterPanic(chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	for _, account := range accounts {
		for blockHash := range account.UnconfirmedBlocks {
			account.deleteAccountBlock(accounts, blockHash)
		}
		account.resetLatestBlock()
	}

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()

	realSnapshotBlocks := snapshotBlockList
	needDeleteSnapshotBlocks := make([]*ledger.SnapshotBlock, 0)
	for i := len(snapshotBlockList) - 1; i >= 0; i-- {
		memSnapshotBlock := snapshotBlockList[i]
		if memSnapshotBlock.Height <= latestSnapshotBlock.Height {
			realSnapshotBlocks = snapshotBlockList[:i+1]
			needDeleteSnapshotBlocks = snapshotBlockList[i+1:]
			break

		}
	}

	for _, account := range accounts {
		account.DeleteSnapshotBlocks(accounts, needDeleteSnapshotBlocks, false)
	}

	return realSnapshotBlocks
}

func saveData(accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) (map[types.Address]*Account, []*ledger.SnapshotBlock) {

	fileName := path.Join(test_tools.DefaultDataDir(), "test_panic")
	fd, oErr := os.OpenFile(fileName, os.O_RDWR, 0666)
	if oErr != nil {
		if os.IsNotExist(oErr) {
			var err error
			fd, err = os.Create(fileName)
			if err != nil {
				panic(err)
			}
		} else {
			panic(oErr)
		}
	}
	if err := fd.Truncate(0); err != nil {
		panic(err)
	}

	if _, err := fd.Seek(0, 0); err != nil {
		panic(err)
	}
	enc := gob.NewEncoder(fd)

	if len(accounts) > 0 {
		if err := enc.Encode(accounts); err != nil {
			panic(err)
		}
	}

	if len(snapshotBlockList) > 0 {
		if err := enc.Encode(snapshotBlockList); err != nil {
			panic(err)
		}
	}

	return accounts, snapshotBlockList
}

func loadData(chainInstance *chain) (map[types.Address]*Account, []*ledger.SnapshotBlock) {
	fileName := path.Join(test_tools.DefaultDataDir(), "test_panic")
	fd, oErr := os.OpenFile(fileName, os.O_RDWR, 0666)
	if oErr != nil {
		if os.IsNotExist(oErr) {
			return make(map[types.Address]*Account), make([]*ledger.SnapshotBlock, 0)
		} else {
			panic(oErr)
		}
	}

	if _, err := fd.Seek(0, 0); err != nil {
		panic(err)
	}

	dec := gob.NewDecoder(fd)
	accounts := make(map[types.Address]*Account)
	dec.Decode(&accounts)

	for _, account := range accounts {
		account.chainInstance = chainInstance
	}

	snapshotList := make([]*ledger.SnapshotBlock, 0)
	dec.Decode(&snapshotList)

	return accounts, snapshotList
}

/**
  fork  rollback only for one forkpoint
*/
func TestChainForkRollBack(t *testing.T) {

	c, accountMap, _ := SetUp(3, 100, 2)
	curSnapshotBlock := c.GetLatestSnapshotBlock()
	fmt.Println(curSnapshotBlock.Height)
	TearDown(c)

	// height
	height := uint64(30)
	fork.SetForkPoints(&config.ForkPoints{
		SeedFork: &config.ForkPoint{
			Height:  height,
			Version: 1,
		},
	})

	c, accountMap, _ = SetUp(10, 0, 0)

	defer func() {
		TearDown(c)
		if err := Clear(c); err != nil {
			t.Fatal(err)
		}
	}()

	curSnapshotBlocknew := c.GetLatestSnapshotBlock()

	fmt.Println(curSnapshotBlocknew.Height, curSnapshotBlocknew.Height == height-1)
	if curSnapshotBlocknew.Height != height-1 {

		t.Fatal(fmt.Sprintf("not equal %+v, %d", curSnapshotBlocknew, height-1))
	}

	InsertAccountBlocks(nil, c, accountMap, 5)

	timeNow := time.Now()
	accountBlockList := c.GetAllUnconfirmedBlocks()

	accountBlockListCopy := make([]*ledger.AccountBlock, 2)
	copy(accountBlockListCopy, accountBlockList[len(accountBlockList)-2:])

	var createSnaoshotContent = func() ledger.SnapshotContent {

		sc := make(ledger.SnapshotContent)

		for i := len(accountBlockList) - 3; i >= 0; i-- {
			if i == len(accountBlockList) {
				continue
			}
			block := accountBlockList[i]
			if _, ok := sc[block.AccountAddress]; !ok {
				sc[block.AccountAddress] = &ledger.HashHeight{
					Hash:   block.Hash,
					Height: block.Height,
				}
			}
		}
		return sc
	}
	sb := &ledger.SnapshotBlock{
		PrevHash:        curSnapshotBlocknew.Hash,
		Height:          curSnapshotBlocknew.Height + 1,
		Timestamp:       &timeNow,
		SnapshotContent: createSnaoshotContent(),
	}
	sb.Hash = sb.ComputeHash()
	delaccountBlockList, err := c.InsertSnapshotBlock(sb)
	if err != nil {
		t.Fatal(err)
	}

	if len(delaccountBlockList) != len(accountBlockListCopy) {
		t.Fatal(fmt.Sprintf("len must be equal %+v, %+v", delaccountBlockList, accountBlockListCopy))

	}
	for index, item := range delaccountBlockList {
		if item.Hash != accountBlockListCopy[index].Hash {
			t.Fatal(fmt.Sprintf("must be equal %+v, %+v", item, accountBlockListCopy[index]))
		}
	}

	accountBlockListNew := c.GetAllUnconfirmedBlocks()
	if len(accountBlockListNew) != 0 {
		t.Fatal(fmt.Sprintf("GetAllUnconfirmedBlocks must be 0, but %d", len(accountBlockListNew)))
	}

}
