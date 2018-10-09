package onroad_test

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/wallet"
	"testing"
	"time"
)

var vite onroad.Vite = testVite{
	chain: chain.NewChain(&config.Config{
		P2P:   nil,
		Miner: nil,
		//Ledger:   nil,
		DataDir:  common.GoViteTestDataDir(),
		FilePort: 0,
		Topo:     nil,
	}),
	wallet: wallet.New(&wallet.Config{
		DataDir: common.GoViteTestDataDir(),
	}),
}

func init() {
	vite.Chain().Init()
	vite.Chain().Start()
}

func generateAddress() types.Address {
	key, _ := vite.WalletManager().KeystoreManager.StoreNewKey("123")
	return key.Address
}

func TestManager_StartAutoReceiveWorker(t *testing.T) {

	addr := generateAddress()

	manager := onroad.NewManager(vite)
	manager.Init()
	manager.StartAutoReceiveWorker(addr, nil)

	time.Sleep(3 * time.Minute)

}
