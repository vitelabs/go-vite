package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"testing"
	"time"
)

func sendViteBlock() (*vm_context.VmAccountBlock, error) {
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

	toAddress, _ := types.HexToAddress("vite_39f1ede9ab4979b8a77167bfade02a3b4df0c413ad048cb999")
	sendAmount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e9))
	var sendBlock = &ledger.AccountBlock{
		PrevHash:       prevHash,
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: ledger.GenesisAccountAddress,
		ToAddress:      toAddress,
		Amount:         sendAmount,
		TokenId:        ledger.ViteTokenId,
		Height:         nextHeight,
		Fee:            big.NewInt(0),
		PublicKey:      publicKey,
		SnapshotHash:   GenesisSnapshotBlock.Hash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.SubBalance(&GenesisMintageSendBlock.TokenId, sendAmount)
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

func TestGetVmLogList(t *testing.T) {
	chainInstance := getChainInstance()
	blocks, err := sendViteBlock()
	if err != nil {
		t.Fatal(err)
	}
	chainInstance.InsertAccountBlocks([]*vm_context.VmAccountBlock{
		blocks,
	})

	logList, err2 := chainInstance.GetVmLogList(blocks.AccountBlock.LogHash)
	if err2 != nil {
		t.Fatal(err2)
	}
	for index, log := range logList {
		fmt.Printf("%d: %+v\n", index, log)
	}
}
