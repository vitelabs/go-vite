package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpcapi/api"
	vite2 "github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func randomSendViteBlock(chainInstance chain.Chain, snapshotBlockHash types.Hash, addr1 *types.Address, addr2 *types.Address) ([]*vm_context.VmAccountBlock, []types.Address, error) {
	now := time.Now()

	if addr1 == nil {
		accountAddress, _, _ := types.CreateAddress()
		addr1 = &accountAddress
	}
	if addr2 == nil {
		accountAddress, _, _ := types.CreateAddress()
		addr2 = &accountAddress
	}

	vmContext, err := vm_context.NewVmContext(chainInstance, nil, nil, addr1)
	if err != nil {
		return nil, nil, err
	}
	latestBlock, _ := chainInstance.GetLatestAccountBlock(addr1)
	nextHeight := uint64(1)
	var prevHash types.Hash
	if latestBlock != nil {
		nextHeight = latestBlock.Height + 1
		prevHash = latestBlock.Hash
	}

	sendAmount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e9))
	var sendBlock = &ledger.AccountBlock{
		PrevHash:       prevHash,
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: *addr2,
		ToAddress:      *addr2,
		Amount:         sendAmount,
		TokenId:        ledger.ViteTokenId,
		Height:         nextHeight,
		Fee:            big.NewInt(0),
		PublicKey:      []byte("public key"),
		SnapshotHash:   snapshotBlockHash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.AddBalance(&ledger.ViteTokenId, sendAmount)

	sendBlock.StateHash = *vmContext.GetStorageHash()
	sendBlock.Hash = sendBlock.ComputeHash()
	return []*vm_context.VmAccountBlock{{
		AccountBlock: sendBlock,
		VmContext:    vmContext,
	}}, []types.Address{*addr1, *addr2}, nil
}

func main() {
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	dataDir := common.HomeDir()
	os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))

	log15.Root().SetHandler(
		log15.LvlFilterHandler(log15.LvlError, log15.Must.FileHandler(filepath.Join(common.DefaultDataDir(), "log"), log15.TerminalFormat())),
	)

	cfg := &config.Config{
		DataDir: dataDir,
		Net: &config.Net{
			Single: true,
		},
		Producer: &config.Producer{},
		Vm:       &config.Vm{IsVmTest: true},
	}
	vite, _ := vite2.New(cfg, wallet.New(&wallet.Config{
		FullSeedStoreFileName: dataDir,
	}))

	vite.Init()
	vite.Start(nil)

	//chainInstance := vite.Chain()
	//os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))
	//chainInstance := chain.NewChain(&config.Config{
	//	FullSeedStoreFileName: common.HomeDir(),
	//Chain: &config.Chain{
	//	KafkaProducers: []*config.KafkaProducer{{
	//		Topic:      "test",
	//		BrokerList: []string{"abc", "def"},
	//	}},
	//},
	//})
	//chainInstance.Init()
	//chainInstance.Start()
	//chainInstance := vite.Chain()
	//addr1, _, _ := types.CreateAddress()
	//addr2, _, _ := types.CreateAddress()
	//blocks, _, _ := randomSendViteBlock(chainInstance, chainInstance.GetGenesisSnapshotBlock().Hash, &addr1, &addr2)

	walletApi := api.NewWalletApi(vite)
	testApi := api.NewTestApi(walletApi)
	var wg sync.WaitGroup

	wg.Add(1)
	selfAddr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	toAddr, _ := types.HexToAddress("vite_39f1ede9ab4979b8a77167bfade02a3b4df0c413ad048cb999")
	if err := testApi.ReceiveOnroadTx(api.CreateReceiveTxParms{
		SelfAddr:   selfAddr,
		FromHash:   chain.GenesisMintageSendBlock.Hash,
		PrivKeyStr: "ab565d7d8819a3548dbdae8561796ccb090692086ff7d5a47eb7b034497cabe73af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a",
	}); err != nil {
		fmt.Printf("CreateTxWithPrivKey failed, error is %s\n", err.Error())
		os.Exit(1)
	}
	go func() {
		defer wg.Done()
		lastTime := time.Now()
		gap := uint64(50000)
		for i := uint64(1); i <= 1000000; i++ {
			if i%gap == 0 {
				now := time.Now()
				ts := uint64(now.Sub(lastTime).Nanoseconds() / 1000)
				fmt.Printf("g1: %d tps\n", (gap*1000*1000)/ts)
				lastTime = time.Now()
			}

			if err := testApi.CreateTxWithPrivKey(api.CreateTxWithPrivKeyParmsTest{
				SelfAddr:    selfAddr,
				ToAddr:      toAddr,
				TokenTypeId: ledger.ViteTokenId,
				PrivateKey:  "ab565d7d8819a3548dbdae8561796ccb090692086ff7d5a47eb7b034497cabe73af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a",
				Amount:      "200000",
				Data:        []byte{0, 2, 3},
				Difficulty:  nil,
			}); err != nil {
				fmt.Printf("CreateTxWithPrivKey failed, error is %v\n", err.Error())
				os.Exit(1)
			}
		}
	}()
	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	//lastTime := time.Now()
	//
	//	for i := uint64(1); i <= 10000000000; i++ {
	//		a := "10"
	//		b := "24"
	//		fmt.Sprintf(a + b)

	//time.Sleep(time.Millisecond * 1)
	//chainInstance.GetAccountBlocksByAddress(&addr1, 0, 1, 10)
	//if i%100000 == 0 {
	//	now := time.Now()
	//	ts := uint64(now.Sub(lastTime).Seconds())
	//	fmt.Printf("g2: %d tps\n", 100000/ts)
	//	lastTime = time.Now()
	//}
	//blocks, _, _ := randomSendViteBlock(chainInstance.GetGenesisSnapshotBlock().Hash, &addr2, &addr1)
	//chainInstance.InsertAccountBlocks(blocks)

	//}
	//}()
	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	lastTime := time.Now()
	//	for i := uint64(1); i <= 10000000; i++ {
	//		if i%100000 == 0 {
	//			now := time.Now()
	//			ts := uint64(now.Sub(lastTime).Seconds())
	//			fmt.Printf("g3: %d tps\n", 100000/ts)
	//			lastTime = time.Now()
	//		}
	//		blocks, _, _ := randomSendViteBlock(chainInstance.GetGenesisSnapshotBlock().Hash, &addr2, &addr1)
	//		chainInstance.InsertAccountBlocks(blocks)
	//
	//	}
	//}()
	wg.Wait()
	blocks, _ := vite.Chain().GetAccountBlocksByAddress(&selfAddr, 0, 1, 1)
	for _, block := range blocks {
		fmt.Printf("%+v\n", block)
	}
}
