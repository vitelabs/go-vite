package verifier

import (
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"os"
	"path/filepath"
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
	vm.InitVmConfig(isTest, false)
}

type VitePrepared struct {
	chain     chain.Chain
	aVerifier *AccountVerifier
	wallet    *wallet.Manager
	onroad    *onroad.Manager
}

func PrepareVite() *VitePrepared {
	dataDir := filepath.Join(common.HomeDir(), "testvite")
	fmt.Printf("----dataDir:%+v\n", dataDir)
	os.RemoveAll(filepath.Join(common.HomeDir(), "ledger"))

	c := chain.NewChain(&config.Config{DataDir: dataDir})

	v := NewAccountVerifier(c, nil)
	w := wallet.New(&wallet.Config{DataDir: dataDir})
	or := onroad.NewManager(nil, nil, nil, w)

	c.Init()
	or.Init(c)
	c.Start()

	//p := pool.NewPool(c)
	//p.Init(&pool.MockSyncer{}, w, nil, v)
	//p.Start()
	return &VitePrepared{
		chain:     c,
		aVerifier: v,
		wallet:    w,
		onroad:    or,
	}
}

//type AddPoolDirect func(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error
type AddChainDierct func(vmAccountBlocks []*vm_context.VmAccountBlock) error

func TestAccountVerifier_VerifyforRPC(t *testing.T) {
	v := PrepareVite()
	if err := verifiyRpcFlow(v, v.chain.InsertAccountBlocks); err != nil {
		t.Error("error", err)
	}
}

func verifiyRpcFlow(vite *VitePrepared, addFunc AddChainDierct) error {
	caseTypeList := []byte{1, 2}
Loop:
	for _, caseType := range caseTypeList {
		var blocks []*ledger.AccountBlock
		var err error
		switch caseType {
		case 0:
			err = GenesisReceiveMintage(vite, addFunc)
			if err == nil {
				// wait next snapshotBlock
				continue
			}
		case 1:
			blocks, err = createRPCTransferBlockS(vite)
		case 2:
			blocks, err = createRPCTransferBlocksR(vite)
		case 3:
			blocks, err = createRPCBlockCallContarct(vite)
		case 4:
			// blocks, err = createRPCBlockCreateContarct(c)
		default:
			break Loop
		}
		if err != nil {
			return err
		}
		if len(blocks) <= 0 {
			return errors.New("createBlock failed")
		}
		for _, b := range blocks {
			vBlocks, err := vite.aVerifier.VerifyforRPC(b)
			if err != nil {
				return err
			}
			if len(vBlocks) > 0 && vBlocks[0] != nil {
				fmt.Printf("blocks[0] balance:%+v,tokenId:%+v\n", vBlocks[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
				if addFunc != nil {
					if err := addFunc(vBlocks); err != nil {
						return errors.New("addFunc failed," + err.Error())
					}
					if caseType == 1 || caseType == 3 {
						if err := vite.onroad.GetOnroadBlocksPool().WriteOnroad(nil, vBlocks); err != nil {
							return errors.New("WriteOnroad failed")
						}
						fmt.Println("addOnroad success")
					}
					fmt.Println("addFunc success")
				}
			} else {
				return errors.New("generator gen an empty block")
			}
		}
	}
	return nil
}

func GenesisReceiveMintage(vite *VitePrepared, addFunc AddChainDierct) error {
	sendBlock, err := vite.chain.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		return err
	}
	fmt.Printf("mintageSend Hash:%+v\n:, toAddress:%+v\n", sendBlock.Hash, sendBlock.ToAddress)

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	gen, err := generator.NewGenerator(vite.chain, nil, nil, &sendBlock.ToAddress)
	if err != nil {
		return err
	}
	genResult, err := gen.GenerateWithOnroad(*sendBlock, nil, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		return ed25519.Sign(genesisAccountPrivKey, data), genesisAccountPubKey, nil
	}, nil)
	if err != nil {
		return err
	}

	blocks := genResult.BlockGenList
	if len(blocks) > 0 && blocks[0] != nil {
		if addFunc != nil {
			if err := addFunc(blocks); err != nil {
				return errors.New("addFunc failed," + err.Error())
			}
			if err := vite.onroad.GetOnroadBlocksPool().WriteOnroad(nil, blocks); err != nil {
				return errors.New("WriteOnroad failed")
			}
			fmt.Println("addFunc success")
		}
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}

func createRPCTransferBlockS(vite *VitePrepared) ([]*ledger.AccountBlock, error) {
	var blocks []*ledger.AccountBlock
	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	latestSb := vite.chain.GetLatestSnapshotBlock()
	latestAb, err := vite.chain.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if latestAb == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("sendBlock's AccountAddress doesn't exist")
	}
	block := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: ledger.GenesisAccountAddress,
		PublicKey:      genesisAccountPubKey,
		ToAddress:      addr1,
		Height:         latestAb.Height + 1,
		PrevHash:       latestAb.Hash,

		Fee:          big.NewInt(0),
		Amount:       big.NewInt(10),
		TokenId:      ledger.ViteTokenId,
		SnapshotHash: latestSb.Hash,
		Timestamp:    latestSb.Timestamp,
	}

	nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	blocks = append(blocks, block)
	return blocks, nil
}

func createRPCTransferBlocksR(vite *VitePrepared) ([]*ledger.AccountBlock, error) {
	var receiveBlocks []*ledger.AccountBlock
	sBlocks, err := vite.onroad.DbAccess().GetAllOnroadBlocks(addr1)
	if err != nil {
		return nil, err
	}
	if len(sBlocks) <= 0 {
		return nil, nil
	}
	for _, v := range sBlocks {
		latestSb := vite.chain.GetLatestSnapshotBlock()
		latestAb, err := vite.chain.GetLatestAccountBlock(&addr1)
		height := uint64(1)
		preHash := types.ZERO_HASH
		if latestAb == nil {
			if err != nil {
				return nil, err
			}
		} else {
			height = latestAb.Height + 1
			preHash = latestAb.Hash
		}
		block := &ledger.AccountBlock{
			BlockType:      ledger.BlockTypeReceive,
			AccountAddress: addr1,
			PublicKey:      addr1PubKey,
			FromBlockHash:  v.Hash,
			Height:         height,
			PrevHash:       preHash,

			Fee:          big.NewInt(0),
			Amount:       v.Amount,
			TokenId:      v.TokenId,
			SnapshotHash: latestSb.Hash,
			Timestamp:    latestSb.Timestamp,
		}

		nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
		block.Nonce = nonce[:]
		block.Hash = block.ComputeHash()
		block.Signature = ed25519.Sign(addr1PrivKey, block.Hash.Bytes())
		receiveBlocks = append(receiveBlocks, block)
	}
	return receiveBlocks, nil
}

func createRPCBlockCallContarct(vite *VitePrepared) ([]*ledger.AccountBlock, error) {
	var blocks []*ledger.AccountBlock

	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()

	// call MethodNamePledge
	pledgeData, _ := contracts.ABIPledge.PackMethod(contracts.MethodNamePledge, addr1)
	latestSb := vite.chain.GetLatestSnapshotBlock()
	latestAb, err := vite.chain.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if latestAb == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("sendBlock's AccountAddress doesn't exist")
	}

	fmt.Printf("latest genesisAccountBlock height：%d, preHash:%+v\n", latestAb.Height, latestAb.PrevHash)
	block := &ledger.AccountBlock{

		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: ledger.GenesisAccountAddress,
		PublicKey:      genesisAccountPubKey,
		ToAddress:      contracts.AddressPledge,
		Amount:         pledgeAmount,
		TokenId:        ledger.ViteTokenId,

		Height:       latestAb.Height + 1,
		PrevHash:     latestAb.Hash,
		Data:         pledgeData,
		Fee:          big.NewInt(0),
		SnapshotHash: latestSb.Hash,
		Timestamp:    latestSb.Timestamp,
	}

	//nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	//block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	blocks = append(blocks, block)
	return blocks, nil
}

func createRPCBlockCreateContarct(c chain.Chain) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func TestAccountVerifier_VerifyforP2P(t *testing.T) {
	v := PrepareVite()
	genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	fromBlock, err := v.chain.GetLatestAccountBlock(&contracts.AddressMintage)
	if err != nil {
		fmt.Println(err)
		return
	}
	latestSb := v.chain.GetLatestSnapshotBlock()
	block := &ledger.AccountBlock{
		Height:         1,
		AccountAddress: ledger.GenesisAccountAddress,
		FromBlockHash:  fromBlock.Hash,
		BlockType:      ledger.BlockTypeReceive,
		Fee:            big.NewInt(0),
		Amount:         big.NewInt(0),
		TokenId:        ledger.ViteTokenId,
		SnapshotHash:   latestSb.Hash,
		Timestamp:      latestSb.Timestamp,
		PublicKey:      genesisAccountPubKey,
	}

	nonce := pow.GetPowNonce(nil, types.DataListHash(block.AccountAddress.Bytes(), block.PrevHash.Bytes()))
	block.Nonce = nonce[:]
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())
	block.Hash = block.ComputeHash()
	block.Signature = ed25519.Sign(genesisAccountPrivKey, block.Hash.Bytes())

	isTrue := v.aVerifier.VerifyNetAb(block)
	t.Log("VerifyforP2P:", isTrue)
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
	}, nil)
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
	v := PrepareVite()
	ts := time.Now()
	block1 := &ledger.AccountBlock{
		Amount:    nil,
		Fee:       nil,
		Hash:      types.Hash{},
		Timestamp: &ts,
	}
	t.Log(v.aVerifier.VerifyDataValidity(block1), block1.Amount.String(), block1.Fee.String())

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
	t.Log(v.aVerifier.VerifyDataValidity(block1), block2.Amount.String(), block2.Fee.String())

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
