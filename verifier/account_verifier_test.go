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
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"math/big"
	"testing"
	"time"
)

var (
	attovPerVite = big.NewInt(1e18)
	pledgeAmount = new(big.Int).Mul(big.NewInt(10), attovPerVite)

	genesisAccountPrivKeyStr string

	addr1, privKey1, _ = types.CreateAddress()
	addr1PrivKey, _    = ed25519.HexToPrivateKey(privKey1.Hex())
	addr1PubKey        = addr1PrivKey.PubByte()

	addr2, _, _ = types.CreateAddress()
	//addr2PrivKey, _    = ed25519.HexToPrivateKey(privKey1.Hex())
	//addr2PubKey        = addr2PrivKey.PubByte()
)

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)
}

func PrepareVite() (chain.Chain, *AccountVerifier) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)
	v := NewAccountVerifier(c, nil, w.KeystoreManager)

	return c, v
}

func TestAccountVerifier_VerifyforRPC(t *testing.T) {
	c, v := PrepareVite()

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

	nonce := pow.GetPowNonce(nil, types.DataHash(append(block.AccountAddress.Bytes(), block.PrevHash.Bytes()...)))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	blocks, err := v.VerifyforRPC(block)
	t.Log(blocks, err)

	t.Log(blocks[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
}

func TestAccountVerifier_VerifyforP2P(t *testing.T) {
	c, v := PrepareVite()
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

	nonce := pow.GetPowNonce(nil, types.DataHash(append(block.AccountAddress.Bytes(), block.PrevHash.Bytes()...)))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	isTrue := v.VerifyforP2P(block)
	t.Log("VerifyforP2P:", isTrue)
}

func TestAccountVerifierFlow(t *testing.T) {
	c, v := PrepareVite()

	// AddressGenesis Receive MintageSend need pow
	mintageSend, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Log("GetLatestAccountBlock", err)
	}
	genResult1, err := AddrGenesisReceiveMintage(c, v, mintageSend)
	if err != nil {
		t.Log("AddrGenesisReceiveMintage", err)
		return
	}
	t.Log("AddrGenesisReceiveMintage result", genResult1)

	// todo CreateNewSnapshotBlock

	// AddressGenesis sendCall PledgeAddress, need pow
	verifyResult1, err := AddrGenesisSendPledge(c, v)
	if err != nil {
		t.Log("AddrGenesisSendPledge", err)
		return
	}
	t.Log("AddrGenesisSendPledge result", verifyResult1)
	t.Log(verifyResult1[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

	// PledgeAddress receive call
	pledgeSend := verifyResult1[0].AccountBlock
	genResult2, err := AddrPledgeReceive(c, v, pledgeSend)
	if err != nil {
		t.Log("AddrGenesisSendPledge", err)
		return
	}
	t.Log("AddrPledgeReceive result", genResult2)

	// test Add1SendAddr2
	verifyResult2, err := Add1SendAddr2(c, v)
	if err != nil {
		t.Log("AddrGenesisSendPledge", err)
		return
	}
	t.Log("Add1SendAddr2 result:", verifyResult2)
}

func AddrGenesisReceiveMintage(c chain.Chain, v *AccountVerifier, sendBlock *ledger.AccountBlock) (*generator.GenResult, error) {
	preAccountBlock, err := c.GetLatestAccountBlock(&sendBlock.ToAddress)
	if err != nil {
		return nil, err
	}
	var preHash *types.Hash
	if preAccountBlock != nil {
		preHash = &preAccountBlock.Hash
	}
	referredSnapshotBlock := c.GetLatestSnapshotBlock()
	gen, err := generator.NewGenerator(v.chain, v.signer, &referredSnapshotBlock.Hash, preHash, &sendBlock.ToAddress)
	if err != nil {
		return nil, err
	}
	return gen.GenerateWithOnroad(*sendBlock, nil, nil)

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
}

func AddrGenesisSendPledge(c chain.Chain, v *AccountVerifier) (blocks []*vm_context.VmAccountBlock, err error) {

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

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

	return v.VerifyforRPC(block)
}

func AddrPledgeReceive(c chain.Chain, v *AccountVerifier, sendBlock *ledger.AccountBlock) (*generator.GenResult, error) {
	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	consensusMessage := &generator.ConsensusMessage{
		SnapshotHash: latestSnapshotBlock.Hash,
		Timestamp:    *latestSnapshotBlock.Timestamp,
		Producer:     types.Address{},
	}

	gen, err := generator.NewGenerator(v.chain, v.signer, &consensusMessage.SnapshotHash, nil, &sendBlock.ToAddress)
	if err != nil {
		return nil, err
	}

	// no sign cause the address isn't unlock
	return gen.GenerateWithOnroad(*sendBlock, consensusMessage, nil)
}

func Add1SendAddr2(c chain.Chain, v *AccountVerifier) (blocks []*vm_context.VmAccountBlock, err error) {

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

	return v.VerifyforRPC(block)
}

func TestAccountVerifier_VerifyDataValidity(t *testing.T) {
	_, v := PrepareVite()
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
