package generator

import (
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
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

func PrepareVite() chain.Chain {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	return c
}

func TestGenerator_GenerateWithOnroad(t *testing.T) {
	c := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Error("GetLatestAccountBlock", err)
		return
	}

	gen, err := NewGenerator(c, nil, nil, &fromBlock.ToAddress)
	if err != nil {
		t.Error(err)
	}
	consensusMsg := &ConsensusMessage{
		SnapshotHash: c.GetLatestSnapshotBlock().Hash,
		Timestamp:    *c.GetLatestSnapshotBlock().Timestamp,
		Producer:     fromBlock.ToAddress,
		gid:          types.Gid{},
	}
	genResult, err := gen.GenerateWithOnroad(*fromBlock, consensusMsg,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			sig := ed25519.Sign(genesisAccountPrivKey, data)
			return sig, genesisAccountPubKey, nil
		})
	if err != nil {
		t.Error("GenerateWithOnroad", err)
		return
	}
	if len(genResult.BlockGenList) > 0 {
		genBlock := genResult.BlockGenList[0].AccountBlock
		mockReceiveBlock := &ledger.AccountBlock{
			Height:         1,
			AccountAddress: ledger.GenesisAccountAddress,
			FromBlockHash:  fromBlock.Hash,
			BlockType:      ledger.BlockTypeReceive,
			Fee:            fromBlock.Fee,
			Amount:         fromBlock.Amount,
			TokenId:        fromBlock.TokenId,
			SnapshotHash:   consensusMsg.SnapshotHash,
			Timestamp:      &consensusMsg.Timestamp,
			PublicKey:      genesisAccountPubKey,
			LogHash:        genBlock.LogHash,
			StateHash:      genBlock.StateHash,
		}
		mockhHash := mockReceiveBlock.ComputeHash()
		t.Log("hash", genBlock.Hash)
		t.Log("mockhBlock", mockhHash)
		if genBlock.Hash != mockhHash {
			t.Log("Verify Hash failed")
			return
		}
		t.Log("Verify Hash success")
	}
}

func TestGenerator_GenerateWithMessage(t *testing.T) {

}
