package compress_test

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"testing"
	"time"
)

var innerChainInstance chain.Chain

func getChainInstance() chain.Chain {
	if innerChainInstance == nil {
		innerChainInstance = chain.NewChain(&config.Config{
			DataDir: common.DefaultDataDir(),
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}
	return innerChainInstance
}

func TestNewCompressor(t *testing.T) {
	chainInstance := getChainInstance()
	compressor := chainInstance.Compressor()

	compressor.Stop()

	compressor.Start()
	compressor.Stop()

	compressor.Start()
	compressor.Stop()
}

func randomViteBlock() (*vm_context.VmAccountBlock, error) {
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

	types.CreateAddress()

	accountAddress, _, _ := types.CreateAddress()
	toAddress, _, _ := types.CreateAddress()

	sendAmount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e9))
	var sendBlock = &ledger.AccountBlock{
		PrevHash:       prevHash,
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: accountAddress,
		ToAddress:      toAddress,
		Amount:         sendAmount,
		TokenId:        ledger.ViteTokenId,
		Height:         nextHeight,
		Fee:            big.NewInt(0),
		PublicKey:      publicKey,
		SnapshotHash:   chain.GenesisSnapshotBlock.Hash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.AddBalance(&chain.GenesisMintageSendBlock.TokenId, sendAmount)
	logHash1, _ := types.HexToHash("1e7f1b0e23a05127e38dca416cf5f4968189e8bd3385c3a1bf554393b0ca8b58")
	logHash2, _ := types.HexToHash("706b00a2ae1725fb5d90b3b7a76d76c922eb075be485749f987af7aa46a66785")
	vmContext.AddLog(&ledger.VmLog{
		Topics: []types.Hash{
			logHash1, logHash2,
		},
		Data: []byte("Yes, I am log"),
	})

	sendBlock.LogHash = vmContext.GetLogListHash()
	sendBlock.StateHash = *vmContext.GetStorageHash()
	sendBlock.Hash = sendBlock.ComputeHash()
	return &vm_context.VmAccountBlock{
		AccountBlock: sendBlock,
		VmContext:    vmContext,
	}, nil
}

func getNewSnapshotBlock() (*ledger.SnapshotBlock, error) {
	chainInstance := getChainInstance()

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

	return snapshotBlock, err
}

func TestRunTask(t *testing.T) {
	chainInstance := getChainInstance()
	compressor := compress.NewCompressor(chainInstance, common.DefaultDataDir())
	compressor.RunTask()

	compressor.Start()
	latestBlock := chainInstance.GetLatestSnapshotBlock()
	if latestBlock.Height < 100000 {
		for i := uint64(0); i < uint64(100000)-latestBlock.Height; i++ {
			block, err := randomViteBlock()
			if err != nil {
				t.Fatal(err)
			}
			chainInstance.InsertAccountBlocks([]*vm_context.VmAccountBlock{block})
			sBlock, err1 := getNewSnapshotBlock()
			if err1 != nil {
				t.Fatal(err1)
			}
			chainInstance.InsertSnapshotBlock(sBlock)
			if i%5000 == 0 {
				fmt.Printf("insert %d snapshotBlock\n", i)
			}
		}
	}

	for i := 0; i < 100; i++ {
		compressor.RunTask()
	}
}

func TestBlockParser(t *testing.T) {
	chainInstance := getChainInstance()
	metas := chainInstance.Compressor().Indexer().Get(200, 3000)
	for _, meta := range metas {
		fileReader := chainInstance.Compressor().FileReader(meta.Filename)
		compress.BlockParser(fileReader, func(block ledger.Block, err error) {
			switch block.(type) {
			case *ledger.AccountBlock:
				fmt.Printf("Ab %d\n", block.(*ledger.AccountBlock).Height)
			case *ledger.SnapshotBlock:
				fmt.Printf("Sb %d\n", block.(*ledger.SnapshotBlock).Height)
			}
		})
	}

}
