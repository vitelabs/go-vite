package verifier

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
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

	producerPrivKeyStr string
)

func init() {
	var isTest bool
	flag.BoolVar(&isTest, "vm.test", false, "test net gets unlimited balance and quota")
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")

	flag.Parse()
	vm.InitVmConfig(isTest)
}

func PrepareVite() (chain.Chain, *AccountVerifier) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	v := NewAccountVerifier(c, nil)
	return c, v
}

func TestAccountVerifier_VerifyforRPC(t *testing.T) {
	c, v := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Error(err)
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

	nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	blocks, err := v.VerifyforRPC(block)
	t.Log(block, err)

	t.Log(blocks[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
}

func TestAccountVerifier_VerifyforRPC_SendCallContract(t *testing.T) {
	c, v := PrepareVite()

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr1)

	block := &ledger.AccountBlock{
		AccountAddress: ledger.GenesisAccountAddress,
		BlockType:      ledger.BlockTypeSendCall,
		ToAddress:      contracts.AddressPledge,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		Data:           pledgeData,
		SnapshotHash:   latestSnapshotBlock.Hash,
		Timestamp:      latestSnapshotBlock.Timestamp,
		PublicKey:      genesisAccountPubKey,
	}

	latestAccountBlock, err := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		t.Error(err)
		return
	}
	if latestAccountBlock != nil {
		block.Height = latestAccountBlock.Height + 1
		block.PrevHash = latestAccountBlock.Hash
	} else {
		block.Height = 1
		block.PrevHash = types.ZERO_HASH
	}

	//nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	//block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	verifyResult1, err := v.VerifyforRPC(block)
	if err != nil {
		t.Error("AddrGenesisSendPledge", err)
		return
	}
	t.Log("AddrGenesisSendPledge result", verifyResult1)
	t.Log(verifyResult1[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
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

	nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	isTrue := v.VerifyNetAb(block)
	t.Log("VerifyforP2P:", isTrue)
}

func TestAccountVerifierFlow(t *testing.T) {
	c, v := PrepareVite()

	// AddressGenesis Receive MintageSend need pow
	mintageSend, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		t.Error("GetLatestAccountBlock", err)
		return
	}
	t.Log("mintageSend:", mintageSend)
	genResult1, err := AddrGenesisReceiveMintage(c, v, mintageSend)
	if err != nil {
		t.Error("AddrGenesisReceiveMintage", err)
		return
	}
	t.Log("AddrGenesisReceiveMintage result", genResult1)

	// todo CreateNewSnapshotBlock

	// AddressGenesis sendCall PledgeAddress, need pow
	verifyResult1, err := AddrGenesisSendPledge(c, v)
	if err != nil {
		t.Error("AddrGenesisSendPledge", err)
		return
	}
	t.Log("AddrGenesisSendPledge result", verifyResult1)
	t.Log(verifyResult1[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)

	// PledgeAddress receive call
	pledgeSend := verifyResult1[0].AccountBlock
	genResult2, err := AddrPledgeReceive(c, v, pledgeSend)
	if err != nil {
		t.Error("AddrGenesisSendPledge", err)
		return
	}
	t.Log("AddrPledgeReceive result", genResult2)

	// test Add1SendAddr2
	verifyResult2, err := Add1SendAddr2(c, v)
	if err != nil {
		t.Error("AddrGenesisSendPledge", err)
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
	gen, err := generator.NewGenerator(v.chain, &referredSnapshotBlock.Hash, preHash, &sendBlock.ToAddress)
	if err != nil {
		return nil, err
	}
	return gen.GenerateWithOnroad(*sendBlock, nil, nil)
}

func AddrGenesisSendPledge(c chain.Chain, v *AccountVerifier) (blocks []*vm_context.VmAccountBlock, err error) {

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr1)

	block := &ledger.AccountBlock{
		AccountAddress: ledger.GenesisAccountAddress,
		BlockType:      ledger.BlockTypeSendCall,
		ToAddress:      contracts.AddressPledge,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		Data:           pledgeData,
		SnapshotHash:   latestSnapshotBlock.Hash,
		Timestamp:      latestSnapshotBlock.Timestamp,
		PublicKey:      genesisAccountPubKey,
	}

	latestAccountBlock, err := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		return nil, err
	}
	if latestAccountBlock != nil {
		block.Height = latestAccountBlock.Height + 1
		block.PrevHash = latestAccountBlock.Hash
	} else {
		block.Height = 1
		block.PrevHash = types.ZERO_HASH
	}

	//nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	//block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	return v.VerifyforRPC(block)
}

func AddrPledgeReceive(c chain.Chain, v *AccountVerifier, sendBlock *ledger.AccountBlock) (*generator.GenResult, error) {
	producerAddress := types.Address{}
	producerPrivKey, _ := ed25519.HexToPrivateKey(producerPrivKeyStr)
	producerPubKey := producerPrivKey.PubByte()

	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	consensusMessage := &generator.ConsensusMessage{
		SnapshotHash: latestSnapshotBlock.Hash,
		Timestamp:    *latestSnapshotBlock.Timestamp,
		Producer:     producerAddress,
	}

	gen, err := generator.NewGenerator(v.chain, &consensusMessage.SnapshotHash, nil, &sendBlock.ToAddress)
	if err != nil {
		return nil, err
	}

	return gen.GenerateWithOnroad(*sendBlock, consensusMessage, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(producerPrivKey, data), producerPubKey, nil
	})
}

func Add1SendAddr2(c chain.Chain, v *AccountVerifier) (blocks []*vm_context.VmAccountBlock, err error) {

	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: addr1,
		PublicKey:      addr1PubKey,
		ToAddress:      addr2,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		SnapshotHash:   latestSnapshotBlock.Hash,
		Timestamp:      latestSnapshotBlock.Timestamp,
	}

	latestAccountBlock, err := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		return nil, err
	}
	if latestAccountBlock != nil {
		block.PrevHash = latestAccountBlock.Hash
		block.Height = latestAccountBlock.Height + 1
	} else {
		block.PrevHash = types.ZERO_HASH
		block.Height = 1
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

func TestAccountVerifier_VerifySigature(t *testing.T) {
	var hashString = "b94ba5fadca89002c48db47b234d15dd0c3f8e39ad68ba74dfb3f63d45b938b6"
	var pubKeyString = "ZTkxYzg2MGM2M2QwMWY2ZDRhOTI5MDE5NzE5MDdhNzIxMDNlOGJhYTg3Zjg4Nzg2NjVmNWE0ZGVmOWVjN2EzNg=="
	var sigString = "MWIyYTUzMTNjYTVjMDAyYTRjODkyYzkwYzIzNDA5NjkyNzM4NjRhMGU1MzhjNDI0MDExODM1NzQ1ZmNiNWY0NjY4ZTc2ZmZiODk5ZjU2Y2Q4ZmU0Y2Q2MjAxODkxMTZkYTgyMmFjMzk5ODU2NDVjNmM0ZTUxMzVhMmNkMTk2MDk="

	pubKey, _ := ed25519.HexToPublicKey(pubKeyString)
	hash, _ := types.HexToHash(hashString)
	sig, _ := hex.DecodeString(sigString)

	if len(sig) == 0 || len(pubKey) == 0 {
		t.Error("sig or pubKey is nil")
		return
	}
	isVerified, verifyErr := crypto.VerifySig(pubKey, hash.Bytes(), sig)
	if !isVerified {
		if verifyErr != nil {
			t.Error("VerifySig failed", "error", verifyErr)
		}
		return
	}
	t.Log("success")
}
