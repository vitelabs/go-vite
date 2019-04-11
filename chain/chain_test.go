package chain

import (
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"testing"

	"encoding/gob"
	"fmt"
	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/quota"
	"math/rand"
)

var genesisConfigJson = "{  \"GenesisAccountAddress\": \"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",  \"ForkPoints\": {  },  \"ConsensusGroupInfo\": {    \"ConsensusGroupInfoMap\":{      \"00000000000000000001\":{        \"NodeCount\":25,        \"Interval\":1,        \"PerCount\":3,        \"RandCount\":2,        \"RandRank\":100,        \"Repeat\":1,        \"CheckLevel\":0,        \"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",        \"RegisterConditionId\":1,        \"RegisterConditionParam\":{          \"PledgeAmount\": 100000000000000000000000,          \"PledgeHeight\": 1,          \"PledgeToken\": \"tti_5649544520544f4b454e6e40\"        },        \"VoteConditionId\":1,        \"VoteConditionParam\":{},        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"PledgeAmount\":0,        \"WithdrawHeight\":1      },      \"00000000000000000002\":{        \"NodeCount\":25,        \"Interval\":3,        \"PerCount\":1,        \"RandCount\":2,        \"RandRank\":100,        \"Repeat\":48,        \"CheckLevel\":1,        \"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",        \"RegisterConditionId\":1,        \"RegisterConditionParam\":{          \"PledgeAmount\": 100000000000000000000000,          \"PledgeHeight\": 1,          \"PledgeToken\": \"tti_5649544520544f4b454e6e40\"        },        \"VoteConditionId\":1,        \"VoteConditionParam\":{},        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"PledgeAmount\":0,        \"WithdrawHeight\":1      }    },    \"RegistrationInfoMap\":{      \"00000000000000000001\":{        \"s1\":{          \"NodeAddr\":\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\",          \"PledgeAddr\":\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\",          \"Amount\":100000000000000000000000,          \"WithdrawHeight\":7776000,          \"RewardTime\":1,          \"CancelTime\":0,          \"HisAddrList\":[\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\"]        },        \"s2\":{          \"NodeAddr\":\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\",          \"PledgeAddr\":\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\",          \"Amount\":100000000000000000000000,          \"WithdrawHeight\":7776000,          \"RewardTime\":1,          \"CancelTime\":0,          \"HisAddrList\":[\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\"]        }      }    }  },  \"MintageInfo\":{    \"TokenInfoMap\":{      \"tti_5649544520544f4b454e6e40\":{        \"TokenName\":\"Vite Token\",        \"TokenSymbol\":\"VITE\",        \"TotalSupply\":1000000000000000000000000000,        \"Decimals\":18,        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"PledgeAmount\":0,        \"PledgeAddr\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"WithdrawHeight\":0,        \"MaxSupply\":115792089237316195423570985008687907853269984665640564039457584007913129639935,        \"OwnerBurnOnly\":false,        \"IsReIssuable\":true      }    },    \"LogList\": [        {          \"Data\": \"\",          \"Topics\": [            \"3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6\",            \"000000000000000000000000000000000000000000005649544520544f4b454e\"          ]        }      ]  },  \"AccountBalanceMap\": {    \"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\": {      \"tti_5649544520544f4b454e6e40\":899999000000000000000000000    },    \"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23\": {      \"tti_5649544520544f4b454e6e40\":100000000000000000000000000    }  }}"
var GenesisJson = "{\"GenesisAccountAddress\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"ForkPoints\":{},\"ConsensusGroupInfo\":{\"ConsensusGroupInfoMap\":{\"00000000000000000001\":{\"NodeCount\":3,\"Interval\":1,\"PerCount\":3,\"RandCount\":2,\"RandRank\":100,\"Repeat\":1,\"CheckLevel\":0,\"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",\"RegisterConditionId\":1,\"RegisterConditionParam\":{\"PledgeAmount\":100000000000000000000000,\"PledgeHeight\":1,\"PledgeToken\":\"tti_5649544520544f4b454e6e40\"},\"VoteConditionId\":1,\"VoteConditionParam\":{},\"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"PledgeAmount\":0,\"WithdrawHeight\":1},\"00000000000000000002\":{\"NodeCount\":3,\"Interval\":3,\"PerCount\":1,\"RandCount\":2,\"RandRank\":100,\"Repeat\":48,\"CheckLevel\":1,\"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",\"RegisterConditionId\":1,\"RegisterConditionParam\":{\"PledgeAmount\":100000000000000000000000,\"PledgeHeight\":1,\"PledgeToken\":\"tti_5649544520544f4b454e6e40\"},\"VoteConditionId\":1,\"VoteConditionParam\":{},\"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"PledgeAmount\":0,\"WithdrawHeight\":1}},\"RegistrationInfoMap\":{\"00000000000000000001\":{\"s1\":{\"NodeAddr\":\"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a\",\"PledgeAddr\":\"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a\",\"Amount\":100000000000000000000000,\"WithdrawHeight\":7776000,\"RewardTime\":1,\"CancelTime\":0,\"HisAddrList\":[\"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a\"]},\"s2\":{\"NodeAddr\":\"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25\",\"PledgeAddr\":\"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25\",\"Amount\":100000000000000000000000,\"WithdrawHeight\":7776000,\"RewardTime\":1,\"CancelTime\":0,\"HisAddrList\":[\"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25\"]},\"s3\":{\"NodeAddr\":\"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a\",\"PledgeAddr\":\"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a\",\"Amount\":100000000000000000000000,\"WithdrawHeight\":7776000,\"RewardTime\":1,\"CancelTime\":0,\"HisAddrList\":[\"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a\"]}}}},\"MintageInfo\":{\"TokenInfoMap\":{\"tti_5649544520544f4b454e6e40\":{\"TokenName\":\"Vite Token\",\"TokenSymbol\":\"VITE\",\"TotalSupply\":1000000000000000000000000000,\"Decimals\":18,\"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"PledgeAmount\":0,\"PledgeAddr\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"WithdrawHeight\":0,\"MaxSupply\":115792089237316195423570985008687907853269984665640564039457584007913129639935,\"OwnerBurnOnly\":false,\"IsReIssuable\":true}},\"LogList\":[{\"Data\":\"\",\"Topics\":[\"3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6\",\"000000000000000000000000000000000000000000005649544520544f4b454e\"]}]},\"PledgeInfo\":{\"PledgeBeneficialMap\":{\"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a\":1000000000000000000000,\"vite_3d544acf90181a0449bb22b4cf02eb4f0bd3ce695da9abc9f8\":1000000000000000000000,\"vite_805f96789429b8ae457a8d1423586a552db7eaa503741e1e8a\":1000000000000000000000,\"vite_6fb5cb4f31d9baa92296920d304055b4d15dd91d08ec3ce344\":1000000000000000000000,\"vite_8a6e2f4d0cbd12be6187dddc5aa00c2deb8b0efc9abb60fb59\":1000000000000000000000,\"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25\":1000000000000000000000,\"vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83\":1000000000000000000000,\"vite_a6059f0227861c3c8c6bb21c8fc85ac9f69d618aeb4f77bc4c\":1000000000000000000000,\"vite_0211767ebbae78be48992bd1d92b9c6fc355e93449862bdaa7\":1000000000000000000000,\"vite_8ec49c82a2a9d1c5ff5a9b84eac8788a9d193ddefc628852cc\":1000000000000000000000,\"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a\":1000000000000000000000,\"vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d\":1000000000000000000000,\"vite_06501648d1d4f6e3f55b6e54d5d4cafa1d22567067e829d860\":1000000000000000000000,\"vite_e82e1bf245292d3f5505f41fbb50b41382b8f562a9cd30ad4e\":1000000000000000000000,\"vite_004d7d2f8f1f18a7d69e2d28a13d0bb1d2c1361b91acf497dc\":1000000000000000000000}},\"AccountBalanceMap\":{\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\":{\"tti_5649544520544f4b454e6e40\":899999000000000000000000000},\"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_3d544acf90181a0449bb22b4cf02eb4f0bd3ce695da9abc9f8\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_805f96789429b8ae457a8d1423586a552db7eaa503741e1e8a\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_6fb5cb4f31d9baa92296920d304055b4d15dd91d08ec3ce344\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_8a6e2f4d0cbd12be6187dddc5aa00c2deb8b0efc9abb60fb59\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_a6059f0227861c3c8c6bb21c8fc85ac9f69d618aeb4f77bc4c\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_0211767ebbae78be48992bd1d92b9c6fc355e93449862bdaa7\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_8ec49c82a2a9d1c5ff5a9b84eac8788a9d193ddefc628852cc\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_06501648d1d4f6e3f55b6e54d5d4cafa1d22567067e829d860\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_e82e1bf245292d3f5505f41fbb50b41382b8f562a9cd30ad4e\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000},\"vite_004d7d2f8f1f18a7d69e2d28a13d0bb1d2c1361b91acf497dc\":{\"tti_5649544520544f4b454e6e40\":100000000000000000000000000}}}"

type MockConsensus struct{}

func (c *MockConsensus) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return true, nil
}

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
	chainInstance.SetConsensus(&MockConsensus{})

	chainInstance.Start()
	return chainInstance, nil
}

func SetUp(t *testing.T, accountNum, txCount, snapshotPerBlockNum int) (*chain, map[types.Address]*Account, []*ledger.SnapshotBlock) {
	// test quota
	quota.InitQuotaConfig(true, true)

	chainInstance, err := NewChainInstance("unit_test", false)
	if err != nil {
		t.Fatal(err)
	}

	InsertSnapshotBlock(chainInstance, true)

	accounts := MakeAccounts(chainInstance, accountNum)
	//unconfirmedBlocks := chainInstance.cache.GetUnconfirmedBlocks()
	//for _, accountBlock := range unconfirmedBlocks {
	//	if _, ok := accounts[accountBlock.AccountAddress]; !ok {
	//		accounts[accountBlock.AccountAddress] = NewAccount(chainInstance, accountBlock.PublicKey, nil)
	//	}
	//}

	//if len(accounts) < accountNum {
	//	lackNum := accountNum - len(accounts)
	//	newAccounts := MakeAccounts(chainInstance, lackNum)
	//	for addr, account := range newAccounts {
	//		accounts[addr] = account
	//	}
	//
	//}
	var snapshotBlockList []*ledger.SnapshotBlock

	t.Run("InsertBlocks", func(t *testing.T) {
		//InsertAccountBlock(t, chainInstance, accounts, txCount, snapshotPerBlockNum)
		snapshotBlockList = InsertAccountBlock(t, chainInstance, accounts, txCount, snapshotPerBlockNum)
	})

	return chainInstance, accounts, snapshotBlockList
}

func TearDown(chainInstance *chain) {
	chainInstance.Stop()
	chainInstance.Destroy()
}

func TestSetup(t *testing.T) {
	SetUp(t, 100, 1231, 9)
}

func TestChain(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	chainInstance, accounts, snapshotBlockList := SetUp(t, 20, 100, 3)
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	snapshotBlockList = append(snapshotBlockList, InsertAccountBlock(t, chainInstance, accounts, rand.Intn(1232), rand.Intn(5))...)

	// test all
	testChainAll(t, chainInstance, accounts, snapshotBlockList)

	// test insert & delete
	snapshotBlockList = testInsertAndDelete(t, chainInstance, accounts, snapshotBlockList)

	// test panic
	//snapshotBlockList = testPanic(t, chainInstance, accounts, snapshotBlockList)

	TearDown(chainInstance)
}

func testChainAll(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	// account
	testAccount(t, chainInstance, accounts)

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

func testPanic(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) []*ledger.SnapshotBlock {
	//cacheFilename := path.Join(test_tools.DefaultDataDir(), "test_panic")
	//gocache := cache.New(0, 0)
	//for addr, account := range accounts {
	//	if err := gocache.Add(addr.String(), account, 0); err != nil {
	//		t.Fatal(err)
	//	}
	//}
	//allItems := gocache.Items()
	//
	//gocache.Save(cacheFilename)
	//gocache = nil
	//
	//gocache2 := cache.New(0, 0)
	//if err := gocache2.LoadFile(cacheFilename); err != nil {
	//	t.Fatal(err)
	//}
	//
	//newAccounts := make(map[types.Address]*Account)
	//for key, item := range allItems {
	//	addr, _ := types.HexToAddress(key)
	//	newAccounts[addr] = item.Object.(*Account)
	//}

	fileName := path.Join(test_tools.DefaultDataDir(), "test_panic")
	fd, oErr := os.OpenFile(fileName, os.O_RDWR, 0666)
	if oErr != nil {
		if os.IsNotExist(oErr) {
			var err error
			fd, err = os.Create(fileName)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatal(oErr)
		}
	}
	if err := fd.Truncate(0); err != nil {
		t.Fatal(err)
	}
	if _, err := fd.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	enc := gob.NewEncoder(fd)
	if err := enc.Encode(accounts); err != nil {
		t.Fatal(err)
	}

	if _, err := fd.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	dec := gob.NewDecoder(fd)
	newAccounts := make(map[types.Address]*Account)
	dec.Decode(&newAccounts)

	for _, account := range newAccounts {
		account.chainInstance = chainInstance
	}
	testChainAll(t, chainInstance, newAccounts, snapshotBlockList)

	for i := 0; i < 10; i++ {
		cmd := reexec.Command("randomPanic")

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		//fmt.Println("main run")
		cmd.Run()
		//if err := cmd.Run(); err != nil {
		//	log.Printf("failed to run command: %s", err)
		//}
		//fmt.Println("main runned")

	}

	return snapshotBlockList

}

func init() {
	reexec.Register("randomPanic", randomPanic)
	if reexec.Init() {
		os.Exit(0)
	}
}

func randomPanic() {

	//var wg sync.WaitGroup
	//for j := 0; j < 3; j++ {
	//	wg.Add(1)
	//	go func() {
	//		defer wg.Done()
	//		// test insert & delete
	//		go func() {
	//			panic("error")
	//		}()
	//
	//		snapshotBlockList = testInsertAndDelete(t, chainInstance, accounts, snapshotBlockList)
	//
	//	}()
	//
	//	wg.Wait()
	//
	//	// recover snapshotBlockList
	//	if len(snapshotBlockList) > 0 {
	//		maxIndex := len(snapshotBlockList) - 1
	//		i := maxIndex
	//		for ; i > 0; i-- {
	//			snapshotBlock := snapshotBlockList[i]
	//			if ok, err := chainInstance.IsSnapshotBlockExisted(snapshotBlock.Hash); err != nil {
	//				t.Fatal(err)
	//			} else if ok {
	//				break
	//			}
	//		}
	//
	//		if i < maxIndex {
	//			for _, account := range accounts {
	//				account.DeleteSnapshotBlocks(accounts, snapshotBlockList[i+1:])
	//			}
	//
	//			snapshotBlockList = snapshotBlockList[:i+1]
	//		}
	//
	//	}
	//
	//	// recover accounts
	//	for _, account := range accounts {
	//		for account.LatestBlock != nil {
	//			if ok, err := chainInstance.IsAccountBlockExisted(account.LatestBlock.Hash); err != nil {
	//				t.Fatal(err)
	//			} else if !ok {
	//				account.rollbackLatestBlock()
	//			} else {
	//				break
	//			}
	//		}
	//	}
	//}

}

func TestChainAcc(t *testing.T) {
	dir := "/Users/liyanda/test_ledger/ledger3"
	c, err := NewChainInstance(dir, false)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	fmt.Println(c.GetLatestSnapshotBlock())
	addr := types.HexToAddressPanic("vite_004d7d2f8f1f18a7d69e2d28a13d0bb1d2c1361b91acf497dc")

	hash, err := types.HexToHash("73b9d44475e2adaed1e19fe78be8b1f9e306ef05dd981e352a40cabda08dbd50")

	block, err := c.GetAccountBlockByHash(hash)
	fmt.Println(block)
	prev, err := c.GetLatestAccountBlock(addr)

	fmt.Println(prev, err)
	prev, err = c.GetAccountBlockByHeight(addr, 5407)
	fmt.Println(prev, err)

	assert.NoError(t, err)
	assert.NotNil(t, prev)
	hashPanic := types.HexToHashPanic("07009804e4a796473b305e6c0b3d1e790a2d31838b410027bb080e8deb6ff361")
	genesis := c.IsGenesisAccountBlock(hashPanic)
	assert.True(t, genesis)
	for i := uint64(1); i <= prev.Height; i++ {
		block, err := c.GetAccountBlockByHeight(addr, i)
		if err != nil {
			panic(err)
		}
		b := c.IsGenesisAccountBlock(block.Hash)

		fmt.Printf("height:%d, producer:%s, hash:%s, %t\n", block.Height, block.Producer(), block.Hash, b)

		//fmt.Printf("%+v\n", block)
	}
}
