package verifier

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
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
	gen := generator.NewGenerator(c, walletManager.KeystoreManager)

	blockTime := time.Now()
	addr, priv, _ := types.CreateAddress()
	fmt.Println(addr.String())
	pubkey := priv.PubByte()

	preBlock, err := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		fmt.Println(err)
		return
	}

	block := &ledger.AccountBlock{
		Height:         preBlock.Height + 1,
		AccountAddress: ledger.GenesisAccountAddress,
		ToAddress:      addr,
		BlockType:      ledger.BlockTypeSendCall,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(1),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   c.GetLatestSnapshotBlock().Hash,
		Timestamp:      &blockTime,
		PublicKey:      pubkey,
		PrevHash:       preBlock.PrevHash,
	}
	verifier := NewAccountVerifier(c, nil)
	verifier.VerifyforRPC(block, gen)
}

func TestAccountVerifier_VerifyDataValidity(t *testing.T) {

}

func TestAccountVerifier_VerifyReferred(t *testing.T) {

}

func TestAccountVerifier_VerifySnapshotBlockOfReferredBlock(t *testing.T) {

}
