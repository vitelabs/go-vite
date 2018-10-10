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
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
	"testing"
)

var (
	genesisAccountPrivKeyStr string
	attovPerVite             = big.NewInt(1e18)
	viteTotalSupply          = new(big.Int).Mul(big.NewInt(1e9), attovPerVite)
	pledgeAmount             = new(big.Int).Mul(big.NewInt(10), attovPerVite)

	addr1, privKey1, _ = types.CreateAddress()
	addr1PrivKey, _    = ed25519.HexToPrivateKey(privKey1.Hex())
	addr1PubKey        = addr1PrivKey.PubByte()
)

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)

}

func TestGenerator_GenerateWithMessage(t *testing.T) {

}

func TestGenerator_PackBlockWithSendBlock(t *testing.T) {
	c, w := PrepareVite()
	gen := NewGenerator(c, w.KeystoreManager)
	fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Error(err)
	}
	gen.PrepareVm(nil, nil, &ledger.GenesisAccountAddress)
	gen.GenerateWithOnroad(*fromBlock, nil, nil)
}

func TestGenerator(t *testing.T) {
	c, w := PrepareVite()

	if err := GenesisBlockMintageReceive(c, w); err != nil {
		t.Log("GenesisBlockMintageReceive", err)
	}

	// todo createNewSb
	CreateNewSnapshotBlock(c, w)

	blocks, err := GenesisBlockPledgeSend(c, w)
	if err != nil {
		t.Log("GenesisBlockPledgeSend", err)
	} else {
		t.Log(blocks, err)
		t.Log(blocks[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
	}

	if err := GenesisBlockPledgeReceive(c, w); err != nil {
		t.Log("GenesisBlockPledgeSend", err)
	}
}

func GenesisBlockMintageReceive(c chain.Chain, w *wallet.Manager) error {
	gen := NewGenerator(c, w.KeystoreManager)
	fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		return err
	}
	gen.PrepareVm(nil, nil, &ledger.GenesisAccountAddress)
	gen.GenerateWithOnroad(*fromBlock, nil, nil)

	//genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	//genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	//fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	//if err != nil {
	//	return err
	//}
	//block := &ledger.AccountBlock{
	//	Height:         1,
	//	AccountAddress: ledger.GenesisAccountAddress,
	//	FromBlockHash:  fromBlock.Hash,
	//	BlockType:      ledger.BlockTypeReceive,
	//	Fee:            big.NewInt(0),
	//	Amount:         big.NewInt(0),
	//	TokenId:        ledger.ViteTokenId,
	//	SnapshotHash:   c.GetLatestSnapshotBlock().Hash,
	//	Timestamp:      c.GetLatestSnapshotBlock().Timestamp,
	//	PublicKey:      genesisAccountPubKey,
	//}
	//
	//nonce := pow.GetPowNonce(nil, types.DataHash(append(block.AccountAddress.Bytes(), block.PrevHash.Bytes()...)))
	//block.Nonce = nonce[:]
	//block.Hash = block.ComputeHash()
	//block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())
	//
	//sendBlock, err := c.GetAccountBlockByHash(&fromBlock.Hash)
	//if err != nil {
	//	return err
	//}
	//gen.generateBlock(block, sendBlock, nil)
	return nil
}

func CreateNewSnapshotBlock(c chain.Chain, w *wallet.Manager) {

}

func GenesisBlockPledgeSend(c chain.Chain, w *wallet.Manager) (blocks []*vm_context.VmAccountBlock, err error) {
	v := verifier.NewAccountVerifier(c, nil)
	g := NewGenerator(c, w.KeystoreManager)

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	latestAccountBlock, _ := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	pledgeData, err := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr1)
	if err != nil {
		return nil, err
	}

	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		Height:         latestAccountBlock.Height + 1,
		ToAddress:      contracts.AddressPledge,
		AccountAddress: ledger.GenesisAccountAddress,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		PrevHash:       latestAccountBlock.Hash,
		Data:           pledgeData,
		SnapshotHash:   latestSnapshotBlock.Hash,
		Timestamp:      latestSnapshotBlock.Timestamp,
		PublicKey:      genesisAccountPubKey,
	}
	nonce := pow.GetPowNonce(nil, types.DataHash(append(block.AccountAddress.Bytes(), block.PrevHash.Bytes()...)))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	return v.VerifyforRPC(block, g)
}

func GenesisBlockPledgeReceive(c chain.Chain, w *wallet.Manager) error {
	//latestAccountBlock, _ := c.GetLatestAccountBlock(&addr1)
	//latestSnapshotBlock := c.GetLatestSnapshotBlock()
	//
	//recvPledgeBlock := &ledger.AccountBlock{
	//	BlockType:      ledger.BlockTypeSendCall,
	//	Height:         0,
	//	PrevHash:       types.Hash{},
	//	AccountAddress: ledger.GenesisAccountAddress,
	//	PublicKey:      genesisAccountPubKey,
	//	ToAddress:      addr1,
	//	FromBlockHash:  types.Hash{},
	//	Amount:         big.NewInt(0),
	//	TokenId:        ledger.ViteTokenId,
	//	Fee:            big.NewInt(0),
	//	SnapshotHash:   latestSnapshotBlock.Hash,
	//	Timestamp:      latestSnapshotBlock.Timestamp,
	//}
	//
	//nonce := pow.GetPowNonce(nil, types.DataHash(append(recvPledgeBlock.AccountAddress.Bytes(), recvPledgeBlock.PrevHash.Bytes()...)))
	//recvPledgeBlock.Nonce = nonce[:]
	//recvPledgeBlock.Hash = recvPledgeBlock.ComputeHash()
	//recvPledgeBlock.Signature = ed25519.Sign(genesisAccountPrivKey, recvPledgeBlock.Hash.Bytes())
	//return nil
}

func TestGenerator_GenerateWithOnroad(t *testing.T) {
}

func PrepareVite() (chain.Chain, *wallet.Manager) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)
	return c, w
}
