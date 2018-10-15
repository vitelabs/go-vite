package generator

import (
	"flag"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
	"testing"
)

var (
	genesisAccountPrivKeyStr string
	addr1, _, _              = types.CreateAddress()

	attovPerVite = big.NewInt(1e18)
	pledgeAmount = new(big.Int).Mul(big.NewInt(10), attovPerVite)
)

func init() {
	var isTest bool
	flag.BoolVar(&isTest, "vm.test", false, "test net gets unlimited balance and quota")
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")

	flag.Parse()
	vm.InitVmConfig(isTest)
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
	genResult, err := gen.GenerateWithOnroad(*fromBlock, nil,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
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

			PublicKey:    genBlock.PublicKey,
			SnapshotHash: genBlock.SnapshotHash,
			Timestamp:    genBlock.Timestamp,
			LogHash:      genBlock.LogHash,
			StateHash:    genBlock.StateHash,
			Nonce:        genBlock.Nonce,
		}
		mockReceiveBlock.Hash = mockReceiveBlock.ComputeHash()
		if genBlock.Hash != mockReceiveBlock.Hash {
			t.Error("Verify Hash failed")
			return
		}
		t.Log("Verify Hash success")
	}
}

func TestGenerator_GenerateWithMessage_CallTransfer(t *testing.T) {
	c := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	message := &IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: ledger.GenesisAccountAddress,
		ToAddress:      &addr1,
		FromBlockHash:  nil,
		TokenId:        &ledger.ViteTokenId,
		Amount:         big.NewInt(10),
		Fee:            nil,
		Data:           nil,
	}

	preBlock, err := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		t.Error("GetLatestAccountBlock", err)
		return
	}
	var preHash *types.Hash
	if preBlock != nil {
		preHash = &preBlock.Hash
		nonce := pow.GetPowNonce(nil, types.DataListHash(ledger.GenesisAccountAddress.Bytes(), preHash.Bytes()))
		message.Nonce = nonce[:]
	} else {
		nonce := pow.GetPowNonce(nil, types.DataListHash(ledger.GenesisAccountAddress.Bytes(), types.ZERO_HASH.Bytes()))
		message.Nonce = nonce[:]
	}

	gen, err := NewGenerator(c, nil, preHash, &message.AccountAddress)
	if err != nil {
		t.Error(err)
		return
	}

	genResult, err := gen.GenerateWithMessage(message, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
	})
	if err != nil {
		t.Error("GenerateWithMessage err", err)
		return
	}
	t.Log("genResult", genResult)
}

func TestGenerator_GenerateWithMessage_CallCompiledContract(t *testing.T) {
	c := PrepareVite()
	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr1)

	im := &IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: ledger.GenesisAccountAddress,
		ToAddress:      &contracts.AddressPledge,
		FromBlockHash:  nil,
		TokenId:        &ledger.ViteTokenId,
		Amount:         pledgeAmount,
		Fee:            nil,
		Nonce:          nil,
		Data:           pledgeData,
	}

	gen, err := NewGenerator(c, nil, nil, &im.AccountAddress)
	if err != nil {
		t.Error(err)
		return
	}

	genResult, err := gen.GenerateWithMessage(im, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
	})
	if err != nil {
		t.Error("GenerateWithMessage err", err)
		return
	}
	t.Log("genResult", genResult)
}

func TestGenerator_GenerateWithBlock(t *testing.T) {

}
