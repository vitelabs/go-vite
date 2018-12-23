package onroad_test

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/vite/net"
	"path/filepath"
	"testing"
	"time"
)

func TestContractWorker_Start(t *testing.T) {
	manager, _ := startManager()
	//worker := onroad.NewContractWorker(manager)

	//event := producerevent.AccountStartEvent{
	//	Gid:            types.DELEGATE_GID,
	//	Address:        addresses,
	//	Stime:          time.Now(),
	//	Etime:          time.Now().Add(time.Minute),
	//	Timestamp:      time.Now(),
	//	SnapshotHash:   types.ZERO_HASH,
	//	SnapshotHeight: 0,
	//}
	//worker.Start(event)

	manager.Producer().(*testProducer).produceEvent(time.Minute)

	time.AfterFunc(5*time.Second, func() {
		fmt.Println("test net sync not complete")
		manager.Net().(*testNet).fn(net.Syncing)
		fmt.Println("end test net not complete")
		time.AfterFunc(20*time.Second, func() {
			fmt.Println("test net sync not Syncdone")
			manager.Net().(*testNet).fn(net.Syncdone)
			fmt.Println("end test net not Syncdone")
		})
	})

	//time.AfterFunc(20*time.Second, func() {
	//	fmt.Println("test stop")
	//	worker.Stop()
	//	worker.Stop()
	//	worker.Stop()
	//
	//	fmt.Println("test stop end")
	//	time.AfterFunc(10*time.Second, func() {
	//		fmt.Println("test stop 1")
	//		worker.Stop()
	//		fmt.Println("test stop end 1")
	//		time.AfterFunc(10*time.Second, func() {
	//			fmt.Println("test Start 1")
	//			worker.Start(event)
	//			fmt.Println("test Start end 1")
	//
	//		})
	//		//fmt.Println("NewOnroadTxAlarm 1")
	//		//worker.NewOnroadTxAlarm()
	//		//time.AfterFunc(10*time.Second, func() {
	//		//	fmt.Println("NewOnroadTxAlarm 2")
	//		//	worker.NewOnroadTxAlarm()
	//		//})
	//	})
	//
	//})

	time.Sleep(5 * time.Minute)
}

func TestOnroadBlocksPool_GetNextContractTx(t *testing.T) {
	db := PrepareDb()
	addr, _ := types.HexToAddress("vite_000000000000000000000000000000000000000270a48cc491")
	p := db.onroad.GetOnroadBlocksPool()
	p.AcquireOnroadSortedContractCache(addr)
	if cList := p.GetContractCallerList(addr); cList != nil {
		t.Logf("cList length", cList.Len())
		for cList.Len() > 0 {
			b := cList.GetNextTx()
			t.Logf("get next: currentCallerIndex=%v fromAddr=%v blockHash=%v height", cList.GetCurrentIndex(), b.AccountAddress, b.Hash, b.Height)
		}
	}
}

type testDb struct {
	chain  chain.Chain
	onroad *onroad.Manager
}

func PrepareDb() *testDb {
	dataDir := filepath.Join(common.HomeDir(), "testvite")
	fmt.Printf("----dataDir:%+v\n", dataDir)
	//os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))

	c := chain.NewChain(&config.Config{DataDir: dataDir})
	or := onroad.NewManager(nil, nil, nil, nil)

	c.Init()
	or.Init(c)
	c.Start()

	return &testDb{
		chain:  c,
		onroad: or,
	}
}
