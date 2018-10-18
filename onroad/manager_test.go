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

var twallet = wallet.New(&wallet.Config{
	DataDir: common.GoViteTestDataDir(),
})

func generateAddress() types.Address {
	key, _ := twallet.KeystoreManager.StoreNewKey("123")
	return key.Address
}

func startManager() (*onroad.Manager, types.Address) {
	addr := generateAddress()

	c := chain.NewChain(&config.Config{
		Net:     nil,
		DataDir: common.GoViteTestDataDir(),
	})
	c.Init()

	prod := new(testProducer)
	prod.Addr = addr

	tnet := new(testNet)

	tpool := new(testPool)

	manager := onroad.NewManager(tnet, tpool, prod, twallet)
	manager.Init(c)

	manager.Start()
	c.Start()

	return manager, addr
}

func TestManager_StartAutoReceiveWorker(t *testing.T) {

	manager, addr := startManager()
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
				twallet.KeystoreManager.Lock(addr)
				time.AfterFunc(5*time.Second, func() {
					fmt.Println("test a unlock ")
					twallet.KeystoreManager.Unlock(addr, "123", 5*time.Second)
					manager.StartAutoReceiveWorker(addr, nil)
				})
			})
		})
	})

	time.Sleep(3 * time.Minute)

}

func TestManager_ContractWorker(t *testing.T) {
	//addr := generateAddress()
	//
	//vite.Producer().(*testProducer).Addr = addr
	//
	//manager := onroad.NewManager(vite)
	//manager.Init()
	//
	//time.AfterFunc(5*time.Second, func() {
	//	fmt.Println("test c produceEvent ")
	//	vite.Producer().(*testProducer).produceEvent(15 * time.Second)
	//})
	//
	//time.Sleep(3 * time.Minute)
}
