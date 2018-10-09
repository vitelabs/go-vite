package net

import (
	"flag"
	"fmt"
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
var needBlockHeight uint64

// -node_name="lyd00" -private_key="xxx" -boot_nodes="xxx,xxx" -make_blocks=true
func init() {
	var bootNodes string

	flag.StringVar(&config.Name, "node_name", "net_test", "server name")
	flag.StringVar(&privateKey, "private_key", "", "server private key")
	flag.StringVar(&bootNodes, "boot_nodes", "", "boot nodes")
	flag.Uint64Var(&needBlockHeight, "need_block_height", 0, "need block height")

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

func makeBlocks(chainInstance chain.Chain, toBlockHeight uint64) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	if latestSnapshotBlock.Height >= toBlockHeight {
		return
	}

	count := latestSnapshotBlock.Height - toBlockHeight

	accountAddress1, _, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()
	for i := uint64(0); i < count; i++ {
		snapshotBlock, _ := newSnapshotBlock(chainInstance)
		chainInstance.InsertSnapshotBlock(snapshotBlock)

		for j := 0; j < 10; j++ {
			blocks, _, _ := randomSendViteBlock(chainInstance, snapshotBlock.Hash, &accountAddress1, &accountAddress2)
			chainInstance.InsertAccountBlocks(blocks)
		}

		if (i+1)%100 == 0 {
			fmt.Printf("Make %d snapshot blocks.\n", i+1)
		}
	}
}

func TestNet(t *testing.T) {
	chainInstance := chain.NewChain(&config2.Config{
		DataDir: common.DefaultDataDir(),
	})

	chainInstance.Init()
	chainInstance.Start()

	if needBlockHeight > 0 {
		makeBlocks(chainInstance, needBlockHeight)
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
