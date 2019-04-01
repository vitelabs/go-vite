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
)

var twallet = wallet.New(&wallet.Config{
	DataDir: common.GoViteTestDataDir(),
})

func generateAddress() types.Address {
	mnemonic, em, _ := twallet.NewMnemonicAndEntropyStore("123456")
	em.Unlock("123456")
	fmt.Println(mnemonic)
	fmt.Println(em.GetEntropyStoreFile())
	fmt.Println(em.GetPrimaryAddr())
	return em.GetPrimaryAddr()
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

/*func TestManager_StartAutoReceiveWorker(t *testing.T) {

	manager, addr := startManager()
	fmt.Println("test chain stop1 ")
	manager.StartAutoReceiveWorker(addr.String(), addr, nil, nil)

	time.AfterFunc(5*time.Second, func() {
		fmt.Println("test chain stop2")
		manager.StopAutoReceiveWorker(addr)
		time.AfterFunc(5*time.Second, func() {
			fmt.Println("test chain start 3")
			manager.StartAutoReceiveWorker(addr.String(), addr, nil, nil)
			time.AfterFunc(5*time.Second, func() {
				fmt.Println("test chain lock4 ")
				storeManager, _ := twallet.GetEntropyStoreManager(addr.String())
				storeManager.Lock()
				//time.AfterFunc(5*time.Second, func() {
				//	fmt.Println("test chain unlock ")
				//	storeManager, _ := twallet.GetEntropyStoreManager(addr.String())
				//	storeManager.Unlock("123456")
				//	manager.StartAutoReceiveWorker(addr.String(), addr, nil)
				//})
			})
		})
	})

	time.Sleep(3 * time.Minute)

}*/

func TestManager_ContractWorker(t *testing.T) {
	/*	addr := generateAddress()

		vite.Producer().(*testProducer).Addr = addr

		manager := onroad.NewManager(vite)
		manager.Init()

		time.AfterFunc(5*time.Second, func() {
			fmt.Println("test c produceEvent ")
			vite.Producer().(*testProducer).produceEvent(15 * time.Second)
		})

		time.Sleep(3 * time.Minute)*/
}
