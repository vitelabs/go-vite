package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func BenchmarkChain_InsertAccountBlocks(b *testing.B) {
	dataDir := common.HomeDir()
	os.RemoveAll(filepath.Join(dataDir, "ledger"))

	//chainInstance := vite.Chain()
	os.RemoveAll(filepath.Join(dataDir, "ledger"))
	chainInstance := NewChain(&config.Config{
		DataDir: common.HomeDir(),
		//Chain: &config.Chain{
		//	KafkaProducers: []*config.KafkaProducer{{
		//		Topic:      "test",
		//		BrokerList: []string{"abc", "def"},
		//	}},
		//},
	})
	chainInstance.Init()
	chainInstance.Start()
	addr1, _, _ := types.CreateAddress()
	addr2, _, _ := types.CreateAddress()
	blocks, _, _ := randomSendViteBlock(chainInstance.GetGenesisSnapshotBlock().Hash, &addr1, &addr2)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		lastTime := time.Now()
		for i := uint64(1); i <= 10000000; i++ {
			if i%300000 == 0 {
				now := time.Now()
				ts := uint64(now.Sub(lastTime).Seconds())
				fmt.Printf("g1: %d tps\n", 300000/ts)
				lastTime = time.Now()
			}
			blocks[0].AccountBlock.Height += 1
			chainInstance.InsertAccountBlocks(blocks)

		}
	}()

	//for i := 0; i < 10; i++ {
	//	wg.Add(1)
	//	go func() {
	//		defer wg.Done()
	//		for i := uint64(1); i <= 10000000000; i++ {
	//			time.Sleep(time.Millisecond * 2)
	//		}
	//	}()
	//}
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
}

func TestContractsAddr(t *testing.T) {
	fmt.Println(contracts.AddressRegister.String())
	fmt.Println(contracts.AddressRegister.String())
	fmt.Println(contracts.AddressRegister.String())
	fmt.Println(contracts.AddressRegister.String())
	fmt.Println(contracts.AddressRegister.String())
}

func TestGetAccountBlocksByHash(t *testing.T) {
	chainInstance := getChainInstance()

	blocks, err1 := chainInstance.GetAccountBlocksByHash(contracts.AddressMintage, nil, 10, true)
	if err1 != nil {
		t.Error(err1)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}

	blocks2, err2 := chainInstance.GetAccountBlocksByHash(contracts.AddressMintage, nil, 10, false)
	if err2 != nil {
		t.Error(err2)
	}
	for index, block := range blocks2 {
		fmt.Printf("%d: %+v\n", index, block)
	}

	startHash, _ := types.HexToHash("f9380deea688b3afe206f52cc3cf2c2677bca1a0fbb4abdfa9d671bc26b22932")
	blocks3, err3 := chainInstance.GetAccountBlocksByHash(contracts.AddressMintage, &startHash, 10, true)
	if err3 != nil {
		t.Error(err3)
	}
	for index, block := range blocks3 {
		fmt.Printf("%d: %+v\n", index, block)
	}

	endHash, _ := types.HexToHash("efe9be9b0e41f37dbb34899bb8891c5e150d35e8e907212128cffb7907b5292a")
	blocks4, err4 := chainInstance.GetAccountBlocksByHash(contracts.AddressMintage, &endHash, 10, false)
	if err4 != nil {
		t.Error(err4)
	}
	for index, block := range blocks4 {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

func TestGetAccountBlockMetaByHash(t *testing.T) {
	chainInstance := getChainInstance()
	hash, _ := types.HexToHash("ad411c4bcafb83ac71b042df3cd4a9f6171ed843cbd51849e93e004b2c379002")
	meta, _ := chainInstance.ChainDb().Ac.GetBlockMeta(&hash)
	fmt.Printf("%+v\n", meta)

}

func TestGetAccountBlocksByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	addr1, _, _ := types.CreateAddress()
	addr2, _, _ := types.CreateAddress()
	for i := 0; i < 100; i++ {
		blocks, _, _ := randomSendViteBlock(chainInstance.GetGenesisSnapshotBlock().Hash, &addr1, &addr2)
		chainInstance.InsertAccountBlocks(blocks)

	}

	blocks, err1 := chainInstance.GetAccountBlocksByHeight(addr1, 1, 1000, true)
	if err1 != nil {
		t.Error(err1)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}

	//blocks2, err2 := chainInstance.GetAccountBlocksByHeight(contracts.AddressMintage, 2, 10, false)
	//if err2 != nil {
	//	t.Error(err2)
	//}
	//for index, block := range blocks2 {
	//	fmt.Printf("%d: %+v\n", index, block)
	//}
	//
	//blocks3, err3 := chainInstance.GetAccountBlocksByHeight(contracts.AddressMintage, 0, 10, true)
	//if err3 != nil {
	//	t.Error(err3)
	//}
	//for index, block := range blocks3 {
	//	fmt.Printf("%d: %+v\n", index, block)
	//}
	//
	//blocks4, err4 := chainInstance.GetAccountBlocksByHeight(contracts.AddressMintage, 1000000, 10, false)
	//if err4 != nil {
	//	t.Error(err4)
	//}
	//for index, block := range blocks4 {
	//	fmt.Printf("%d: %+v\n", index, block)
	//}
}

func TestChain_GetAccountBlockMap(t *testing.T) {
	chainInstance := getChainInstance()
	startHash, _ := types.HexToHash("f9380deea688b3afe206f52cc3cf2c2677bca1a0fbb4abdfa9d671bc26b22932")
	queryParams1 := map[types.Address]*BlockMapQueryParam{
		contracts.AddressMintage: {
			OriginBlockHash: &startHash,
			Count:           10,
			Forward:         true,
		},
		contracts.AddressConsensusGroup: {
			OriginBlockHash: nil,
			Count:           10,
			Forward:         true,
		},
		contracts.AddressRegister: {
			OriginBlockHash: nil,
			Count:           10,
			Forward:         true,
		},
	}

	blockMap := chainInstance.GetAccountBlockMap(queryParams1)

	for addr, blocks := range blockMap {
		fmt.Println(addr.String())
		for index, block := range blocks {
			fmt.Printf("%d: %+v\n", index, block)
		}
	}

	fmt.Println()
	queryParams2 := map[types.Address]*BlockMapQueryParam{
		contracts.AddressMintage: {
			OriginBlockHash: nil,
			Count:           10,
			Forward:         false,
		},
		contracts.AddressConsensusGroup: {
			OriginBlockHash: nil,
			Count:           10,
			Forward:         false,
		},
		contracts.AddressRegister: {
			OriginBlockHash: nil,
			Count:           10,
			Forward:         false,
		},
	}

	blockMap2 := chainInstance.GetAccountBlockMap(queryParams2)

	for addr, blocks := range blockMap2 {
		fmt.Println(addr.String())
		for index, block := range blocks {
			fmt.Printf("%d: %+v\n", index, block)
		}
	}
}

func receiveViteBlock() (*vm_context.VmAccountBlock, error) {
	chainInstance := getChainInstance()
	publicKey, _ := ed25519.HexToPublicKey("3af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a")
	now := time.Now()
	vmContext, err := vm_context.NewVmContext(chainInstance, nil, nil, &ledger.GenesisAccountAddress)
	if err != nil {
		return nil, err
	}

	latestBlock, _ := chainInstance.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	nextHeight := uint64(1)
	var prevHash types.Hash
	if latestBlock != nil {
		nextHeight = latestBlock.Height + 1
		prevHash = latestBlock.Hash
	}

	var receiveBlock = &ledger.AccountBlock{
		PrevHash:       prevHash,
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: ledger.GenesisAccountAddress,
		FromBlockHash:  GenesisMintageSendBlock.Hash,
		Height:         nextHeight,
		PublicKey:      publicKey,
		SnapshotHash:   GenesisSnapshotBlock.Hash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.AddBalance(&GenesisMintageSendBlock.TokenId, GenesisMintageSendBlock.Amount)
	receiveBlock.StateHash = *vmContext.GetStorageHash()
	receiveBlock.Hash = receiveBlock.ComputeHash()
	return &vm_context.VmAccountBlock{
		AccountBlock: receiveBlock,
		VmContext:    vmContext,
	}, nil

}

func TestGetAccountBalance(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := receiveViteBlock()
	if err != nil {
		t.Fatal(err)
	}
	err2 := chainInstance.InsertAccountBlocks([]*vm_context.VmAccountBlock{block})
	if err2 != nil {
		t.Fatal(err)
	}
	balanceMap, err3 := chainInstance.GetAccountBalance(&block.AccountBlock.AccountAddress)
	if err3 != nil {
		t.Fatal(err3)
	}

	fmt.Printf("%+v\n", balanceMap)

	balance, err4 := chainInstance.GetAccountBalanceByTokenId(&block.AccountBlock.AccountAddress, &GenesisMintageSendBlock.TokenId)
	if err4 != nil {
		t.Fatal(err4)
	}

	fmt.Printf("%+v\n", balance)

	blocks, err1 := chainInstance.GetAccountBlocksByHeight(block.AccountBlock.AccountAddress, 1, 10, true)
	if err1 != nil {
		t.Error(err1)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

func TestGetAccountBlockHashByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	hash, err := chainInstance.GetAccountBlockHashByHeight(&ledger.GenesisAccountAddress, 1)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(hash.String())
}

func TestGetAccountBlockByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	addr, _ := types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")
	block, err := chainInstance.GetAccountBlockByHeight(&addr, 4239)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%+v\n", block)
}

func TestGetAccountBlockByHash(t *testing.T) {
	chainInstance := getChainInstance()
	//hash, _ := types.HexToHash("8df6b1e1f3a9574016a7853dd90fb39c21d8d4aacedbb600b405b3d984b2f5c4")
	hash := types.Hash{}
	block, err := chainInstance.GetAccountBlockByHash(&hash)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%+v\n", block)

	//fromBlock, err2 := chainInstance.GetAccountBlockByHash(&GenesisMintageSendBlock.Hash)
	//if err2 != nil {
	//	t.Error(err2)
	//}
	//fmt.Printf("%+v\n", fromBlock)
	//fmt.Printf("%+v\n", fromBlock.Meta)
}

func TestGetAccountBlocksByAddress(t *testing.T) {
	chainInstance := getChainInstance()
	blocks, err := chainInstance.GetAccountBlocksByAddress(&ledger.GenesisAccountAddress, 0, 1, 15)
	if err != nil {
		t.Error(err)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

func TestGetFirstConfirmedAccountBlockBySbHeight(t *testing.T) {
	chainInstance := getChainInstance()
	addr, _ := types.HexToAddress("vite_5acd0b2ef651bdc0c586aafe7a780103f45ac532cd886eb859")
	block, err := chainInstance.GetFirstConfirmedAccountBlockBySbHeight(19943, &addr)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%+v\n", block)

	//block2, err2 := chainInstance.GetFirstConfirmedAccountBlockBySbHeight(3, &ledger.GenesisAccountAddress)
	//if err2 == nil {
	//	t.Error("Test error")
	//}
	//fmt.Printf("%+v\n", block2)
	//
	//block3, err3 := chainInstance.GetFirstConfirmedAccountBlockBySbHeight(1, &ledger.GenesisAccountAddress)
	//if err3 != nil {
	//	t.Error(err3)
	//}
	//fmt.Printf("%+v\n", block3)
	//
	//block4, err4 := chainInstance.GetFirstConfirmedAccountBlockBySbHeight(1, &GenesisMintageSendBlock.AccountAddress)
	//if err4 != nil {
	//	t.Error(err4)
	//}
	//fmt.Printf("%+v\n", block4)
	//
	//block5, err5 := chainInstance.GetFirstConfirmedAccountBlockBySbHeight(2, &GenesisMintageSendBlock.AccountAddress)
	//if err5 != nil {
	//	t.Error(err5)
	//}
	//fmt.Printf("%+v\n", block5)
}

func TestGetUnConfirmAccountBlocks(t *testing.T) {
	chainInstance := getChainInstance()
	blocks := chainInstance.GetUnConfirmAccountBlocks(&ledger.GenesisAccountAddress)
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}

	fmt.Println()
	blocks2 := chainInstance.GetUnConfirmAccountBlocks(&GenesisMintageSendBlock.AccountAddress)
	for index, block := range blocks2 {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

// TODO need snapshot
func TestDeleteAccountBlocks(t *testing.T) {
	chainInstance := getChainInstance()

	snapshotBlock, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock)

	blocks, addressList, _ := randomSendViteBlock(snapshotBlock.Hash, nil, nil)
	chainInstance.InsertAccountBlocks(blocks)

	blocks2, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks2)

	blocks3, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks3)

	snapshotBlock2, _ := newSnapshotBlock()
	chainInstance.InsertSnapshotBlock(snapshotBlock2)

	blocks4, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks4)

	blocks5, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks5)

	blocks6, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks6)

	blocks7, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addressList[0], &addressList[1])
	chainInstance.InsertAccountBlocks(blocks7)

	receiveBlock, _ := newReceiveBlock(snapshotBlock.Hash, addressList[1], blocks[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock)

	receiveBlock2, _ := newReceiveBlock(snapshotBlock.Hash, addressList[1], blocks2[0].AccountBlock.Hash)
	chainInstance.InsertAccountBlocks(receiveBlock2)

	var display = func() {
		dBlocks1, _ := chainInstance.GetAccountBlocksByHeight(blocks[0].AccountBlock.AccountAddress, 0, 10, true)
		for _, block := range dBlocks1 {
			fmt.Printf("%+v\n", block)
		}

		dBlocks3, _ := chainInstance.GetAccountBlocksByHeight(receiveBlock[0].AccountBlock.AccountAddress, 0, 10, true)
		for _, block := range dBlocks3 {
			fmt.Printf("%+v\n", block)
		}

		latestBlock := chainInstance.GetLatestSnapshotBlock()
		fmt.Printf("%+v\n", latestBlock)

	}
	display()
	fmt.Println()
	snapshotContent := chainInstance.GetNeedSnapshotContent()
	for addr, hashHeight := range snapshotContent {
		fmt.Printf("addr is %s\n", addr.String())
		fmt.Printf("hash is %s, height is %d\n", hashHeight.Hash.String(), hashHeight.Height)
	}
	fmt.Println()

	deleteSubLedger, err := chainInstance.DeleteAccountBlocks(&blocks[0].AccountBlock.AccountAddress, 4)
	if err != nil {
		t.Fatal(err)
	}
	for addr, blocks := range deleteSubLedger {
		fmt.Printf("addr is %s\n", addr.String())
		for _, block := range blocks {
			fmt.Printf("%v\n", block)
		}
	}
	fmt.Println()
	display()
	fmt.Println()

	snapshotContent2 := chainInstance.GetNeedSnapshotContent()
	for addr, hashHeight := range snapshotContent2 {
		fmt.Printf("addr is %s\n", addr.String())
		fmt.Printf("hash is %s, height is %d\n", hashHeight.Hash.String(), hashHeight.Height)
	}
	fmt.Println()
}
