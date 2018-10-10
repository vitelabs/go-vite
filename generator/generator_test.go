package generator

import (
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/wallet"
	"testing"
)

var (
	genesisAccountPrivKeyStr string
)

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)
}

func PrepareVite() (chain.Chain, *wallet.Manager) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)
	return c, w
}

func TestGenerator_GenerateWithOnroad(t *testing.T) {
	c, w := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Error("GetLatestAccountBlock", err)
		return
	}

	gen, err := NewGenerator(c, w.KeystoreManager, nil, nil, &fromBlock.ToAddress)
	if err != nil {
		t.Error(err)
	}
	genResult, err := gen.GenerateWithOnroad(*fromBlock, nil, nil)
	if err != nil {
		t.Error("GenerateWithOnroad", err)
		return
	}
	if len(genResult.BlockGenList) > 0 {
		block := &ledger.AccountBlock{
			Height:         1,
			AccountAddress: ledger.GenesisAccountAddress,
			FromBlockHash:  fromBlock.Hash,
			BlockType:      ledger.BlockTypeReceive,
			Fee:            fromBlock.Fee,
			Amount:         fromBlock.Amount,
			TokenId:        fromBlock.TokenId,
			SnapshotHash:   c.GetLatestSnapshotBlock().Hash,
			Timestamp:      c.GetLatestSnapshotBlock().Timestamp,
			PublicKey:      genesisAccountPubKey,
		}
		mockhHash := block.ComputeHash()
		if genResult.BlockGenList[0].AccountBlock.Hash == mockhHash {
			t.Log("Verify Hash success")
			return
		}
	}
}

func TestGenerator_GenerateWithMessage(t *testing.T) {

}
