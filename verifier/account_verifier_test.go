package verifier

import (
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
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

func PrepareVite() (chain.Chain, *generator.Generator, *AccountVerifier) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)
	g := generator.NewGenerator(c, w.KeystoreManager)

	v := NewAccountVerifier(c, nil)

	return c, g, v
}

func TestAccountVerifier_VerifyforRPC(t *testing.T) {
	c, g, v := PrepareVite()
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
		Timestamp:      c.GetLatestSnapshotBlock().Timestamp,
		PublicKey:      genesisAccountPubKey,
	}
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	blocks, err := v.VerifyforRPC(block, g)
	t.Log(blocks, err)

	balanceMap, err := c.GetAccountBalance(&ledger.GenesisAccountAddress)
	t.Log(balanceMap, err)
}

func TestAccountVerifier_VerifyforP2P(t *testing.T) {
	c, _, v := PrepareVite()
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
		Timestamp:      c.GetLatestSnapshotBlock().Timestamp,
		PublicKey:      genesisAccountPubKey,
	}
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())
	isTrue := v.VerifyforP2P(block)
	t.Log("VerifyforP2P:", isTrue)
}

func TestAccountVerifier_VerifyDataValidity(t *testing.T) {
	_, _, v := PrepareVite()
	ts := time.Now()
	block1 := &ledger.AccountBlock{
		Amount:    nil,
		Fee:       nil,
		Hash:      types.Hash{},
		Timestamp: &ts,
	}
	t.Log(v.VerifyDataValidity(block1), block1.Amount.String(), block1.Fee.String())

	// verifyHash
	block2 := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: contracts.AddressPledge,
		Amount:         big.NewInt(100),
		Fee:            big.NewInt(100),
		Timestamp:      nil,
		Signature:      nil,
		PublicKey:      nil,
	}
	t.Log(v.VerifyDataValidity(block1), block2.Amount.String(), block2.Fee.String())

	// verifySig

	return
}

func TestAccountVerifier_VerifySnapshotBlockOfReferredBlock(t *testing.T) {

}

func TestAccountVerifier_VerifyReferred(t *testing.T) {
	c, g, v := PrepareVite()
	TestverifyFrom(c, g, v, t)

}

func TestverifyFrom(c chain.Chain, g *generator.Generator, v *AccountVerifier, t *testing.T) {
	stat := v.newVerifyStat()
	block := &ledger.AccountBlock{}
	v.verifyFrom(block, stat)
	return
}

func TestAccountVerifier_verifySelf(t *testing.T) {
	c, g, v := PrepareVite()
	TestverifySelfPrev(c, g, v, t)
}

func TestverifySelfPrev(c chain.Chain, g *generator.Generator, v *AccountVerifier, t *testing.T) {
	stat := v.newVerifyStat()
	block := &ledger.AccountBlock{}
	vr, err := v.verifySelfPrev(block, stat.accountTask)
	if err != nil {
		t.Error(err)
	}
	t.Log("verifier result", vr)
}
