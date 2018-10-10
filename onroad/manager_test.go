package onroad_test

import (
	"fmt"
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
	producer: new(testProducer),
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

	vite.Producer().(*testProducer).Addr = addr

	manager := onroad.NewManager(vite)
	manager.Init()
	fmt.Println("test a stop ")
	manager.StartAutoReceiveWorker(addr, nil)

	time.AfterFunc(5*time.Second, func() {
		fmt.Println("test a stop ")
		manager.StopAutoReceiveWorker(addr)
		time.AfterFunc(5*time.Second, func() {
			fmt.Println("test a start ")
			manager.StartAutoReceiveWorker(addr, nil)
			time.AfterFunc(5*time.Second, func() {
				fmt.Println("test a lock ")
				vite.WalletManager().KeystoreManager.Lock(addr)
				time.AfterFunc(5*time.Second, func() {
					fmt.Println("test a unlock ")
					vite.WalletManager().KeystoreManager.Unlock(addr, "123", 5*time.Second)
					manager.StartAutoReceiveWorker(addr, nil)
				})
			})
		})
	})


	time.Sleep(3 * time.Minute)

}

func TestManager_ContractWorker(t *testing.T) {
	addr := generateAddress()

	vite.Producer().(*testProducer).Addr = addr

	manager := onroad.NewManager(vite)
	manager.Init()

	time.AfterFunc(5*time.Second, func() {
		fmt.Println("test c produceEvent ")
		vite.Producer().(*testProducer).produceEvent(15 * time.Second)
	})

	time.Sleep(3 * time.Minute)
}
