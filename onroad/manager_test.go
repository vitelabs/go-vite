package onroad

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/wallet"
	"os"
	"path"
	"testing"
	"time"
)

var (
	unitTestPath      = "unit_test/devdata"
	genesisConfigJSON = "{  \"GenesisAccountAddress\": \"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",  \"ForkPoints\": {  },  \"SeedFork \":{}, \"GovernanceInfo\": {    \"ConsensusGroupInfoMap\":{      \"00000000000000000001\":{        \"NodeCount\":25,        \"Interval\":1,        \"PerCount\":3,        \"RandCount\":2,        \"RandRank\":100,        \"Repeat\":1,        \"CheckLevel\":0,        \"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",        \"RegisterConditionId\":1,        \"RegisterConditionParam\":{          \"StakeAmount\": 100000000000000000000000,          \"StakeHeight\": 1,          \"StakeToken\": \"tti_5649544520544f4b454e6e40\"        },        \"VoteConditionId\":1,        \"VoteConditionParam\":{},        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"StakeAmount\":0,        \"ExpirationHeight\":1      },      \"00000000000000000002\":{        \"NodeCount\":25,        \"Interval\":3,        \"PerCount\":1,        \"RandCount\":2,        \"RandRank\":100,        \"Repeat\":48,        \"CheckLevel\":1,        \"CountingTokenId\":\"tti_5649544520544f4b454e6e40\",        \"RegisterConditionId\":1,        \"RegisterConditionParam\":{          \"StakeAmount\": 100000000000000000000000,          \"StakeHeight\": 1,          \"StakeToken\": \"tti_5649544520544f4b454e6e40\"        },        \"VoteConditionId\":1,        \"VoteConditionParam\":{},        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"StakeAmount\":0,        \"ExpirationHeight\":1      }    },    \"RegistrationInfoMap\":{      \"00000000000000000001\":{        \"s1\":{          \"BlockProducingAddress\":\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\",          \"StakeAddress\":\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\",          \"Amount\":100000000000000000000000,          \"ExpirationHeight\":7776000,          \"RewardTime\":1,          \"RevokeTime\":0,          \"HistoryAddressList\":[\"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08\"]        },        \"s2\":{          \"BlockProducingAddress\":\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\",          \"StakeAddress\":\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\",          \"Amount\":100000000000000000000000,          \"ExpirationHeight\":7776000,          \"RewardTime\":1,          \"RevokeTime\":0,          \"HistoryAddressList\":[\"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87\"]        }      }    }  },  \"AssetInfo\":{    \"TokenInfoMap\":{      \"tti_5649544520544f4b454e6e40\":{        \"TokenName\":\"Vite Token\",        \"TokenSymbol\":\"VITE\",        \"TotalSupply\":1000000000000000000000000000,        \"Decimals\":18,        \"Owner\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"StakeAmount\":0,        \"StakeAddress\":\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",        \"ExpirationHeight\":0,        \"MaxSupply\":115792089237316195423570985008687907853269984665640564039457584007913129639935,        \"IsOwnerBurnOnly\":false,        \"IsReIssuable\":true      }    },    \"LogList\": [        {          \"Data\": \"\",          \"Topics\": [            \"3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6\",            \"000000000000000000000000000000000000000000005649544520544f4b454e\"          ]        }      ]  },  \"AccountBalanceMap\": {    \"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\": {      \"tti_5649544520544f4b454e6e40\":899999000000000000000000000    },    \"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23\": {      \"tti_5649544520544f4b454e6e40\":100000000000000000000000000    }  }}"
	tWallet           = wallet.New(&wallet.Config{DataDir: path.Join(common.HomeDir(), unitTestPath, "wallet")})
)

func generateUnlockAddress() types.Address {
	mnemonic, em, _ := tWallet.NewMnemonicAndEntropyStore("1")
	em.Unlock("1")
	fmt.Println(mnemonic)
	fmt.Println(em.GetEntropyStoreFile())
	fmt.Println(em.GetPrimaryAddr())
	return em.GetPrimaryAddr()
}

func NewChainInstance(dirName string, clear bool) (chain.Chain, error) {
	dataDir := path.Join(common.HomeDir(), dirName)
	fmt.Println("dataDir is", dataDir)
	if clear {
		os.RemoveAll(dataDir)
	}
	if len(fork.GetActiveForkPointList()) <= 0 {
		fork.SetForkPoints(&config.ForkPoints{
			SeedFork: &config.ForkPoint{
				Version: 1,
				Height:  10000000,
			},
		})
	}
	genesisConfig := &config.Genesis{}
	json.Unmarshal([]byte(genesisConfigJSON), genesisConfig)
	chainInstance := chain.NewChain(dataDir, &config.Chain{}, genesisConfig)
	if err := chainInstance.Init(); err != nil {
		return nil, err
	}
	chainInstance.Start()
	return chainInstance, nil
}

func TestManager_ContractWorker(t *testing.T) {
	chainInstance, err := NewChainInstance(unitTestPath, true)
	if err != nil {
		t.Fatal(err)
	}
	v := NewMockVite(chainInstance)

	addr := generateUnlockAddress()
	v.Producer().(*mockProducer).Addr = addr

	manager := NewManager(v.Net(), v.Pool(), v.Producer(), nil, tWallet)
	manager.Init(v.chain)
	manager.Start()

	time.AfterFunc(5*time.Second, func() {
		t.Log("test c produceEvent ")
		v.Producer().(*mockProducer).produceEvent(15 * time.Second)
	})

	time.Sleep(30 * time.Second)
	manager.Stop()
}
