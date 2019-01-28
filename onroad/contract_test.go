package onroad_test

import (
	"encoding/base64"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
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
	sb := db.chain.GetLatestSnapshotBlock()
	if sb == nil {
		return
	}
	t.Logf("snapshotbloclk: hash %v, height %v ", sb.Hash, sb.Height)
	addr, _ := types.HexToAddress("vite_000000000000000000000000000000000000000270a48cc491")
	//blocks, err := db.onroad.DbAccess().GetAllOnroadBlocks(addr)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//t.Logf("totalnum %v", len(blocks))

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

func TestAutoReceiveWorker_Status(t *testing.T) {
	c := chain.NewChain(&config.Config{DataDir: "/Users/crzn/Library/GVite/devdata"})

	w := wallet.New(nil)

	p := pool.NewPool(c)
	cs := &consensus.MockConsensus{}
	av := verifier.NewAccountVerifier(c, cs)
	sv := verifier.NewSnapshotVerifier(c, cs)
	p.Init(&pool.MockSyncer{}, w, sv, av)
	p.Start()

	or := onroad.NewManager(nil, p, nil, w)

	c.Init()
	or.Init(c)
	c.Start()

	addrstr1 := ""
	addrstr2 := ""
	_, err := types.HexToAddress(addrstr1)
	if err != nil {
		t.Fatal(err)
	}
	addr2, err := types.HexToAddress(addrstr2)
	if err != nil {
		t.Fatal(err)
	}
	privByte, err := base64.StdEncoding.DecodeString("")
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Log(common.Bytes2Hex(privByte))

	voteDateBase64 := "/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3NzcwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	data, err := base64.StdEncoding.DecodeString(voteDateBase64)
	if err != nil {
		t.Fatal(err)
		return
	}

	db := PrepareDb()
	var privkey ed25519.PrivateKey
	privkey = privByte
	tokenId, err := types.HexToTokenTypeId("tti_8973a8d5ba05430c6821f3ff")
	if err != nil {
		t.Fatal(err)
	}
	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: addr2,
		ToAddress:      &types.AddressPledge,
		TokenId:        &tokenId,
		Amount:         big.NewInt(10000),
		Data:           data,
	}
	_, fitestSnapshotBlockHash, err := generator.GetFittestGeneratorSnapshotHash(db.chain, &msg.AccountAddress, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	g, e := generator.NewGenerator(db.chain, fitestSnapshotBlockHash, nil, &msg.AccountAddress)
	if e != nil {
		t.Fatal(e)
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		signData := ed25519.Sign(privkey, data)
		pubkey = privkey.PubByte()
		return signData, pubkey, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		if err := p.AddDirectAccountBlock(addr2, result.BlockGenList[0]); err != nil {
			t.Error(err)
		}
	} else {
		t.Error("generator gen an empty block")
		return
	}
}
