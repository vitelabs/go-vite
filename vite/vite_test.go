package vite

import (
	"flag"
	"fmt"
	"math/big"
	"testing"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var genesisAccountPrivKeyStr string

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)

}

func PrepareVite() (chain.Chain, *verifier.AccountVerifier, pool.BlockPool) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)

	v := verifier.NewAccountVerifier(c, nil)

	p := pool.NewPool(c)

	p.Init(&pool.MockSyncer{}, w, verifier.NewSnapshotVerifier(c, nil), v)
	p.Start()

	return c, v, p
}

func TestSend(t *testing.T) {
	c, v, p := PrepareVite()
	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		fmt.Println(err)
		return
	}
	block := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: ledger.GenesisAccountAddress,
		FromBlockHash:  fromBlock.Hash,
		BlockType:      ledger.BlockTypeReceive,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   c.GetLatestSnapshotBlock().Hash,
		Timestamp:      c.GetLatestSnapshotBlock().Timestamp,
		PublicKey:      genesisAccountPubKey,
	}

	nonce := pow.GetPowNonce(nil, types.DataHash(append(block.AccountAddress.Bytes(), block.PrevHash.Bytes()...)))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	blocks, err := v.VerifyforRPC(block)
	t.Log(blocks, err)

	t.Log(blocks[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

	t.Log(blocks[0].AccountBlock.Hash)

	p.AddDirectAccountBlock(ledger.GenesisAccountAddress, blocks[0])

	accountBlock, e := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)

	t.Log(accountBlock.Hash, e)
}

//func TestNew(t *testing.T) {
//	config := &config.Config{
//		DataDir: common.DefaultDataDir(),
//		Producer: &config.Producer{
//			Producer: false,
//		},
//	}
//
//	w := wallet.New(nil)
//	vite, err := New(config, w)
//	if err != nil {
//		t.Error(err)
//		return
//	}
//	err = vite.Init()
//	if err != nil {
//		t.Error(err)
//		return
//	}
//	err = vite.Start()
//	if err != nil {
//		t.Error(err)
//		return
//	}
//}

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

func BenchmarkChain_InsertAccountBlocks(b *testing.B) {
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
	vite, _ := New(cfg, wallet.New(&wallet.Config{
		DataDir: dataDir,
	}))

	vite.Init()
	vite.Start(nil)

	//chainInstance := vite.Chain()
	//os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))
	//chainInstance := chain.NewChain(&config.Config{
	//	DataDir: common.HomeDir(),
	//Chain: &config.Chain{
	//	KafkaProducers: []*config.KafkaProducer{{
	//		Topic:      "test",
	//		BrokerList: []string{"abc", "def"},
	//	}},
	//},
	//})
	//chainInstance.Init()
	//chainInstance.Start()
	chainInstance := vite.Chain()
	addr1, _, _ := types.CreateAddress()
	addr2, _, _ := types.CreateAddress()
	blocks, _, _ := randomSendViteBlock(chainInstance, chainInstance.GetGenesisSnapshotBlock().Hash, &addr1, &addr2)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		lastTime := time.Now()
		for i := uint64(1); i <= 10000000; i++ {
			if i%200000 == 0 {
				now := time.Now()
				ts := uint64(now.Sub(lastTime).Nanoseconds() / 1000)
				fmt.Printf("g1: %d tps\n", (200000*1000*1000)/ts)
				lastTime = time.Now()
			}
			blocks[0].AccountBlock.Height += 1
			chainInstance.InsertAccountBlocks(blocks)

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
}
