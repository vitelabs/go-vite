package verifier

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
	"testing"
	"time"
)

func TestAccountVerifier_VerifyforRPC(t *testing.T) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()
	walletManager := wallet.New(nil)

	verifier := NewAccountVerifier(c, nil)
	gen := generator.NewGenerator(c, walletManager.KeystoreManager)
	blockTime := time.Now()
	block := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: addr1,
		ToAddress:      addr2,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(1),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   snapshot1.Hash,
		Timestamp:      &blockTime,
	}
	verifier.VerifyforVM(block, gen)
}

func TestAccountVerifier_VerifyforVM(t *testing.T) {

}

func TestAccountVerifier_VerifyReferred(t *testing.T) {

}

func TestAccountVerifier_VerifySnapshotBlockOfReferredBlock(t *testing.T) {

}
