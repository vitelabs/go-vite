package chain

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

var genesisConfigJson = "{  \"GenesisAccountAddress\": \"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",  \"ForkPoints\": {  },  \"ConsensusGroupInfo\": {    \"ConsensusGroupInfoMap\":{      \"00000000000000000001\":{        \"NodeCount\":25,        \"Interval\":1,        \"PerCount\":3,        \"RandCount\":2,        \"RandRank\":100,        \"Repeat\":1,        \"CheckLevel\":0,        \"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",        \"RegisterConditionId\":1,        \"RegisterConditionParam\":{          \"PledgeAmount\": 100000000000000000000000,          \"PledgeHeight\": 1,          \"PledgeToken\": \"tti_5649544520544f4b454e6e40\"        },        \"VoteConditionId\":1,        \"VoteConditionParam\":{},        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"PledgeAmount\":0,        \"WithdrawHeight\":1      },      \"00000000000000000002\":{        \"NodeCount\":25,        \"Interval\":3,        \"PerCount\":1,        \"RandCount\":2,        \"RandRank\":100,        \"Repeat\":48,        \"CheckLevel\":1,        \"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",        \"RegisterConditionId\":1,        \"RegisterConditionParam\":{          \"PledgeAmount\": 100000000000000000000000,          \"PledgeHeight\": 1,          \"PledgeToken\": \"tti_5649544520544f4b454e6e40\"        },        \"VoteConditionId\":1,        \"VoteConditionParam\":{},        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"PledgeAmount\":0,        \"WithdrawHeight\":1      }    },    \"RegistrationInfoMap\":{      \"00000000000000000001\":{        \"s1\":{          \"NodeAddr\":\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\",          \"PledgeAddr\":\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\",          \"Amount\":100000000000000000000000,          \"WithdrawHeight\":7776000,          \"RewardTime\":1,          \"CancelTime\":0,          \"HisAddrList\":[\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\"]        },        \"s2\":{          \"NodeAddr\":\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\",          \"PledgeAddr\":\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\",          \"Amount\":100000000000000000000000,          \"WithdrawHeight\":7776000,          \"RewardTime\":1,          \"CancelTime\":0,          \"HisAddrList\":[\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\"]        }      }    }  },  \"MintageInfo\":{    \"TokenInfoMap\":{      \"tti_5649544520544f4b454e6e40\":{        \"TokenName\":\"Vite Token\",        \"TokenSymbol\":\"VITE\",        \"TotalSupply\":1000000000000000000000000000,        \"Decimals\":18,        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"PledgeAmount\":0,        \"PledgeAddr\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"WithdrawHeight\":0,        \"MaxSupply\":115792089237316195423570985008687907853269984665640564039457584007913129639935,        \"OwnerBurnOnly\":false,        \"IsReIssuable\":true      }    },    \"LogList\": [        {          \"Data\": \"\",          \"Topics\": [            \"3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6\",            \"000000000000000000000000000000000000000000005649544520544f4b454e\"          ]        }      ]  },  \"AccountBalanceMap\": {    \"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\": {      \"tti_5649544520544f4b454e6e40\":899999000000000000000000000    },    \"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23\": {      \"tti_5649544520544f4b454e6e40\":100000000000000000000000000    }  }}"

func NewChainInstance(dirName string, clear bool) (*chain, error) {
	dataDir := path.Join(defaultDataDir(), dirName)
	if clear {
		os.RemoveAll(dataDir)
	}
	genesisConfig := &config.Genesis{}
	json.Unmarshal([]byte(genesisConfigJson), genesisConfig)

	chainInstance := NewChain(dataDir, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		return nil, err
	}
	chainInstance.Start()
	return chainInstance, nil
}

func defaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "GVite")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "GVite")
		} else {
			return filepath.Join(home, ".gvite")
		}
	}
	// As we cannot guess chain stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func SetUp(t *testing.T, accountNum, txCount, snapshotPerBlockNum int) (*chain, map[types.Address]*Account, []types.Hash, []types.Address, []uint64, []*ledger.SnapshotBlock) {
	chainInstance, err := NewChainInstance("unit_test", false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Insert Blocks")

	var accounts map[types.Address]*Account
	var hashList []types.Hash
	var addrList []types.Address
	var heightList []uint64
	var snapshotBlockList []*ledger.SnapshotBlock

	t.Run("InsertBlocks", func(t *testing.T) {
		accounts, hashList, addrList, heightList, snapshotBlockList = InsertAccountBlock(t, accountNum, chainInstance, txCount, snapshotPerBlockNum)
	})

	return chainInstance, accounts, hashList, addrList, heightList, snapshotBlockList
}

func TearDown(chainInstance *chain) {
	chainInstance.Stop()
	chainInstance.Destroy()
}

func TestChain(t *testing.T) {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	chainInstance, accounts, hashList, addrList, heightList, snapshotBlockList := SetUp(t, 1000, 200000, 198)

	// account
	testAccount(t, chainInstance, addrList)

	// account block
	testAccountBlock(t, chainInstance, accounts, hashList, addrList, heightList)

	// on road
	testOnRoad(t, chainInstance, accounts, addrList)

	// snapshot block
	testSnapshotBlock(t, chainInstance, accounts, snapshotBlockList)

	// state
	testState(t, chainInstance, accounts)

	// built-in contract
	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)

	TearDown(chainInstance)

}
