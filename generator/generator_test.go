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
	attovPerVite    = big.NewInt(1e18)
	viteTotalSupply = new(big.Int).Mul(big.NewInt(1e9), attovPerVite)
	pledgeAmount    = new(big.Int).Mul(big.NewInt(10), attovPerVite)

	genesisAccountPrivKeyStr string
	genesisAccountPrivKey, _ = ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey     = genesisAccountPrivKey.PubByte()

	addr1, privKey1, _ = types.CreateAddress()
	addr1PrivKey, _    = ed25519.HexToPrivateKey(privKey1.Hex())
	addr1PubKey        = addr1PrivKey.PubByte()

	addr2, privKey2, _ = types.CreateAddress()
	addr2PrivKey, _    = ed25519.HexToPrivateKey(privKey1.Hex())
	addr2PubKey        = addr2PrivKey.PubByte()
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

func TestGeneratorFlow(t *testing.T) {
	c, w := PrepareVite()

	// AddressGenesis Receive MintageSend need pow
	mintageSend, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Log("GetLatestAccountBlock", err)
	}
	if err := AddrGenesisReceiveMintage(&c, w, mintageSend); err != nil {
		t.Log("AddrGenesisReceiveMintage", err)
	}

	CreateNewSnapshotBlock(&c, w)

	// AddressGenesis sendCall PledgeAddress, need pow
	verifyResult, err := AddrGenesisSendPledge(&c, w)
	if err != nil {
		t.Log("AddrGenesisSendPledge", err)
		return
	}
	t.Log(verifyResult)
	t.Log(verifyResult[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

	// PledgeAddress receive call
	pledgeSend := verifyResult[0].AccountBlock
	genResult, err := AddrPledgeReceive(c, w, pledgeSend)
	if err != nil {
		t.Log("AddrGenesisSendPledge", err)
	}

	// test Add1SendAddr2
	Add1SendAddr2(c, w)
	t.Log(genResult)
}

func AddrGenesisReceiveMintage(c chain.Chain, w *wallet.Manager, sendBlock *ledger.AccountBlock) error {
	gen := NewGenerator(c, w.KeystoreManager)

	gen.PrepareVm(nil, nil, &ledger.GenesisAccountAddress)
	gen.GenerateWithOnroad(*sendBlock, nil, nil)

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

func AddrGenesisSendPledge(c chain.Chain, w *wallet.Manager) (blocks []*vm_context.VmAccountBlock, err error) {
	v := verifier.NewAccountVerifier(c, nil)
	g := NewGenerator(c, w.KeystoreManager)

	latestAccountBlock, _ := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr1)
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

func AddrPledgeReceive(c chain.Chain, w *wallet.Manager, sendBlock *ledger.AccountBlock) (*GenResult, error) {
	g := NewGenerator(c, w.KeystoreManager)

	var preHash types.Hash
	latestAccountBlock, err := c.GetLatestAccountBlock(&contracts.AddressPledge)
	if err != nil {
		return nil, err
	}

	if latestAccountBlock != nil {
		preHash = latestAccountBlock.Hash
	}
	latestSnapshotBlock := c.GetLatestSnapshotBlock()

	consensusMessage := &ConsensusMessage{
		SnapshotHash: latestSnapshotBlock.Hash,
		Timestamp:    *latestSnapshotBlock.Timestamp,
		Producer:     types.Address{},
		gid:          types.Gid{},
	}
	g.PrepareVm(&consensusMessage.SnapshotHash, &preHash, &contracts.AddressPledge)

	// no sign cause the address isn't unlock
	return g.GenerateWithOnroad(*sendBlock, consensusMessage, nil)
}

func Add1SendAddr2(c chain.Chain, w *wallet.Manager) (blocks []*vm_context.VmAccountBlock, err error) {
	v := verifier.NewAccountVerifier(c, nil)
	g := NewGenerator(c, w.KeystoreManager)

	var preHash types.Hash
	var height uint64 = 1
	latestAccountBlock, err := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		return nil, err
	}
	if latestAccountBlock != nil {
		preHash = latestAccountBlock.Hash
		height = height + 1
	}
	latestSnapshotBlock := c.GetLatestSnapshotBlock()

	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: addr1,
		PublicKey:      addr1PubKey,
		ToAddress:      addr2,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		PrevHash:       preHash,
		Height:         height,
		SnapshotHash:   latestSnapshotBlock.Hash,
		Timestamp:      latestSnapshotBlock.Timestamp,
	}

	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(addr1PrivKey, block.Hash.Bytes())

	return v.VerifyforRPC(block, g)
}

func TestGenerator_GenerateWithMessage(t *testing.T) {

}
