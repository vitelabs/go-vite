package net

import (
	"flag"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	config2 "github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strings"
	"testing"
	"time"
)

var config = p2p.Config{
	Name:            "",
	NetID:           10,
	MaxPeers:        100,
	MaxPendingPeers: 20,
	MaxInboundRatio: 3,
	Port:            8483,
	PrivateKey:      nil,
	BootNodes:       nil,
}

var privateKey string
var needMakeBlocks bool

func init() {
	var bootNodes string

	flag.StringVar(&config.Name, "name", "net_test", "server name")
	flag.StringVar(&privateKey, "private key", "", "server private key")
	flag.StringVar(&bootNodes, "boot nodes", "", "boot nodes")
	flag.BoolVar(&needMakeBlocks, "make blocks", false, "whether need make blocks")

	config.BootNodes = strings.Split(bootNodes, ",")

	flag.Parse()
}
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
		AccountAddress: *addr1,
		ToAddress:      *addr2,
		Amount:         sendAmount,
		TokenId:        ledger.ViteTokenId,
		Height:         nextHeight,
		Fee:            big.NewInt(0),
		//PublicKey:      publicKey,
		SnapshotHash: snapshotBlockHash,
		Timestamp:    &now,
		Nonce:        []byte("test nonce test nonce"),
		Signature:    []byte("test signature test signature test signature"),
	}

	vmContext.AddBalance(&ledger.ViteTokenId, sendAmount)

	sendBlock.StateHash = *vmContext.GetStorageHash()
	sendBlock.Hash = sendBlock.ComputeHash()
	return []*vm_context.VmAccountBlock{{
		AccountBlock: sendBlock,
		VmContext:    vmContext,
	}}, []types.Address{*addr1, *addr2}, nil
}

func newSnapshotBlock(chainInstance chain.Chain) (*ledger.SnapshotBlock, error) {

	latestBlock := chainInstance.GetLatestSnapshotBlock()
	now := time.Now()
	snapshotBlock := &ledger.SnapshotBlock{
		Height:    latestBlock.Height + 1,
		PrevHash:  latestBlock.Hash,
		Timestamp: &now,
	}

	content := chainInstance.GetNeedSnapshotContent()
	snapshotBlock.SnapshotContent = content

	trie, err := chainInstance.GenStateTrie(latestBlock.StateHash, content)
	if err != nil {
		return nil, err
	}

	snapshotBlock.StateTrie = trie
	snapshotBlock.StateHash = *trie.Hash()
	snapshotBlock.Hash = snapshotBlock.ComputeHash()

	return snapshotBlock, err
}

func makeBlocks(chainInstance chain.Chain) {
	accountAddress1, _, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()
	for i := 0; i < 1000000; i++ {
		snapshotBlock, _ := newSnapshotBlock(chainInstance)
		chainInstance.InsertSnapshotBlock(snapshotBlock)

		for j := 0; j < 10; j++ {
			blocks, _, _ := randomSendViteBlock(chainInstance, snapshotBlock.Hash, &accountAddress1, &accountAddress2)
			chainInstance.InsertAccountBlocks(blocks)
		}
	}
}

func TestNet(t *testing.T) {
	chainInstance := chain.NewChain(&config2.Config{
		DataDir: common.DefaultDataDir(),
	})

	chainInstance.Init()
	chainInstance.Start()

	if needMakeBlocks {
		makeBlocks(chainInstance)
	}

	net, err := New(&Config{
		Port:     8484,
		Chain:    chainInstance,
		Verifier: nil,
	})

	if err != nil {
		t.Error(err)
	}

	svr, err := p2p.New(config)
	if err != nil {
		t.Error(err)
	}

	svr.Protocols = append(svr.Protocols, net.Protocols...)

	err = svr.Start()
	if err != nil {
		t.Error(err)
	}
}
