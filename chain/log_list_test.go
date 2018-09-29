package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
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
		Height:         nextHeight,
		PublicKey:      publicKey,
		SnapshotHash:   GenesisSnapshotBlock.Hash,
		Timestamp:      &now,
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	vmContext.SubBalance(&GenesisMintageSendBlock.TokenId, sendAmount)
	sendBlock.StateHash = *vmContext.GetStorageHash()
	sendBlock.Hash = sendBlock.ComputeHash()
	return &vm_context.VmAccountBlock{
		AccountBlock: sendBlock,
		VmContext:    vmContext,
	}, nil
}

//
//func TestGetVmLogList() {
//
//}
