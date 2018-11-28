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
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

var (
	genesisAccountPrivKeyStr string
	addr1, privKey1, _       = types.CreateAddress()
	addr1PrivKey, _          = ed25519.HexToPrivateKey(privKey1.Hex())
	addr1PubKey              = addr1PrivKey.PubByte()
	addr2, _, _              = types.CreateAddress()

	attovPerVite = big.NewInt(1e18)
	pledgeAmount = new(big.Int).Mul(big.NewInt(10), attovPerVite)
	// difficulty test:65535~67108863
	defaultDifficulty = big.NewInt(65535)
)

func init() {
	var isTest bool
	flag.BoolVar(&isTest, "vm.test", false, "test net gets unlimited balance and quota")
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")

	flag.Parse()
	vm.InitVmConfig(isTest, false)
}

type VitePrepared struct {
	chain chain.Chain
}

func PrepareVite() *VitePrepared {
	dataDir := filepath.Join(common.HomeDir(), "testvite")
	fmt.Printf("----dataDir:%+v\n", dataDir)
	os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))

	c := chain.NewChain(&config.Config{DataDir: dataDir})
	c.Init()
	c.Start()

	return &VitePrepared{
		chain: c,
	}
}

func TestGenerator_GenerateWithOnroad(t *testing.T) {
	v := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	fromBlock, err := v.chain.GetLatestAccountBlock(&types.AddressMintage)
	if err != nil {
		t.Error("GetLatestAccountBlock", err)
		return
	}
	var referredSnapshotHashList []types.Hash
	referredSnapshotHashList = append(referredSnapshotHashList, fromBlock.SnapshotHash)
	_, fittestSnapshotBlockHash, err := GetFittestGeneratorSnapshotHash(v.chain, &fromBlock.ToAddress, referredSnapshotHashList, true)
	if err != nil {
		t.Error("GetFittestGeneratorSnapshotHash", err)
		return
	}
	gen, err := NewGenerator(v.chain, fittestSnapshotBlockHash, nil, &fromBlock.ToAddress)
	if err != nil {
		t.Error(err)
	}
	genResult, err := gen.GenerateWithOnroad(*fromBlock, nil,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
		}, defaultDifficulty)
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

func TestGenerator_GenerateWithMessage_CallCompiledContract(t *testing.T) {
	v := PrepareVite()
	if err := createRPCBlockCallPledgeContarct(v, &addr1); err != nil {
		t.Error(err)
		return
	}
}

func createRPCBlockCallPledgeContarct(vite *VitePrepared, addr *types.Address) error {
	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	// call MethodNamePledge
	pledgeData, _ := abi.ABIPledge.PackMethod(abi.MethodNamePledge, addr)

	im := &IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: ledger.GenesisAccountAddress,
		ToAddress:      &types.AddressPledge,
		Amount:         pledgeAmount,
		TokenId:        &ledger.ViteTokenId,
		Data:           pledgeData,
	}

	_, fittestSnapshotBlockHash, err := GetFittestGeneratorSnapshotHash(vite.chain, &im.AccountAddress, nil, true)
	if err != nil {
		return err
	}
	gen, err := NewGenerator(vite.chain, fittestSnapshotBlockHash, nil, &im.AccountAddress)
	if err != nil {
		return err
	}

	genResult, err := gen.GenerateWithMessage(im, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("blocks[0] balance:%+v,tokenId:%+v\n", genResult.BlockGenList[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
	return nil
}

func TestGenerator_GenerateWithMessage_CallTransfer(t *testing.T) {
	v := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	if err := callTransfer(v, &ledger.GenesisAccountAddress, &addr1, genesisAccountPrivKey, genesisAccountPubKey, defaultDifficulty); err != nil {
		t.Error(err)
		return
	}
	if err := callTransfer(v, &addr1, &addr2, addr1PrivKey, addr1PubKey, defaultDifficulty); err != nil {
		t.Error(err)
		return
	}
}

func callTransfer(vite *VitePrepared, fromAddr, toAddr *types.Address, fromAddrPrivKey ed25519.PrivateKey, fromAddrPubKey []byte, difficulty *big.Int) error {
	im := &IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: *fromAddr,
		ToAddress:      toAddr,
		Amount:         big.NewInt(10),
		TokenId:        &ledger.ViteTokenId,
		Difficulty:     difficulty,
	}

	_, fittestSnapshotBlockHash, err := GetFittestGeneratorSnapshotHash(vite.chain, &im.AccountAddress, nil, true)
	if err != nil {
		return err
	}
	gen, err := NewGenerator(vite.chain, fittestSnapshotBlockHash, nil, &im.AccountAddress)
	if err != nil {
		return err
	}

	genResult, err := gen.GenerateWithMessage(im, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(fromAddrPrivKey, data), fromAddrPubKey, nil
	})
	if err != nil {
		return err
	}
	if genResult.Err != nil {
		fmt.Printf("genResult.Err:%v\n", genResult.Err)
	}
	//fmt.Printf("genResult.BlockGenList:%v\n", genResult.BlockGenList)
	fmt.Printf("blocks[0] balance:%+v,tokenId:%+v\n", genResult.BlockGenList[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
	return nil
}

func TestGenerator(t *testing.T) {
	//gen, _ := NewGenerator(nil,nil, nil, nil)
	gen := &Generator{log: log15.New()}
	block, err := gen.generateBlock(&ledger.AccountBlock{}, &ledger.AccountBlock{}, types.Address{}, nil)
	t.Error(err, block)
}
