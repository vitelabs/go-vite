package verifier

import (
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
	"testing"
	"time"
)

var genesisAccountPrivKeyStr string

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)
}

func TestAccountVerifier_VerifyforRPC(t *testing.T) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	walletManager := wallet.New(nil)
	gen := generator.NewGenerator(c, walletManager.KeystoreManager)

	blockTime := time.Now()
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
		Timestamp:      &blockTime,
		PublicKey:      genesisAccountPubKey,
	}
	verifier := NewAccountVerifier(c, nil)
	blocks, err := verifier.VerifyforRPC(block, gen)
	t.Log(blocks, err)

	balanceMap, err := c.GetAccountBalance(&ledger.GenesisAccountAddress)
	t.Log(balanceMap, err)
}

func TestAccountVerifier_VerifyDataValidity(t *testing.T) {

}

func TestAccountVerifier_VerifyReferred(t *testing.T) {

}

func TestAccountVerifier_VerifySnapshotBlockOfReferredBlock(t *testing.T) {

}
