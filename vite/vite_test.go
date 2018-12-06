package vite

import (
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"strings"
	"testing"

	_ "net/http/pprof"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/hexutil"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/wallet"
)

var genesisAccountPrivKeyStr string

func init() {
	flag.StringVar(&genesisAccountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(genesisAccountPrivKeyStr)

}

func PrepareVite() (chain.Chain, *verifier.AccountVerifier, pool.BlockPool) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	w := wallet.New(nil)

	v := verifier.NewAccountVerifier(c, nil)

	p := pool.NewPool(c)

	p.Init(&pool.MockSyncer{}, w, verifier.NewSnapshotVerifier(c, nil), v)
	p.Start()

	return c, v, p
}

func TestSend(t *testing.T) {
	//c, v, p := PrepareVite()
	//genesisAccountPrivKey, _ := ed25519.HexToPrivateKey(genesisAccountPrivKeyStr)
	//genesisAccountPubKey := genesisAccountPrivKey.PubByte()
	//fromBlock, err := c.GetLatestAccountBlock(&contracts.AddressMintage)
	//if err != nil {
	//	fmt.Println(err)
	//	return
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
	//blocks, err := v.VerifyforRPC(block)
	//t.Log(blocks, err)
	//
	//t.Log(blocks[0].VmContext.GetBalance(&ledger.GenesisAccountAddress, &ledger.ViteTokenId), err)
	//
	//t.Log(blocks[0].AccountBlock.Hash)
	//
	//p.AddDirectAccountBlock(ledger.GenesisAccountAddress, blocks[0])
	//
	//accountBlock, e := c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	//
	//t.Log(accountBlock.Hash, e)
}

//func TestNew(t *testing.T) {
//	config := &config.Config{
//		DataDir: common.DefaultDataDir(),
//		Producer: &config.Producer{
//			Producer: false,
//		},
//	}
//
//	w := wallet.New(nil)
//	vite, err := New(config, w)
//	if err != nil {
//		t.Error(err)
//		return
//	}
//	err = vite.Init()
//	if err != nil {
//		t.Error(err)
//		return
//	}
//	err = vite.Start()
//	if err != nil {
//		t.Error(err)
//		return
//	}
//}
func TestSplit(t *testing.T) {
	addresses, u, e := parseCoinbase("1:vite_91dc0c38d104c7915d3a6c4381a40c360edd871c34ac255bb2")
	if e != nil {
		t.Error(e)
	}
	t.Log(addresses)
	t.Log(u)
}

func TestGen(t *testing.T) {
	abiString := `[{"constant":false,"inputs":[{"name":"body","type":"uint256[]"}],"name":"transfer","outputs":[],"payable":true,"stateMutability":"payable","type":"function"}]`
	abiContract, _ := abi.JSONToABIContract(strings.NewReader(abiString))
	inputString := "vite_f92993a570df8582b9e677dc75c480ec7beebacebcd82cf672,400" +
		",vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304,600" +
		",vite_228f578d58842437fb52104b25750aa84a6f8558b6d9e970b1,200" +
		",vite_562d82b129541fc4ceb6cca3fec234b09dbaa3412cf248e298,100" +
		",vite_e1cf2c438d2e88a8beb49f887f823f36bbe66f724f11e4a4e7,300"
	intputStrList := strings.Split(inputString, ",")
	list := make([]*big.Int, len(intputStrList))
	for i, inputStr := range intputStrList {
		if len(inputStr) == 55 && inputStr[:5] == "vite_" {
			list[i], _ = new(big.Int).SetString(inputStr[5:45], 16)
		} else {
			list[i], _ = new(big.Int).SetString(inputStr, 10)
		}
	}
	data, err := abiContract.PackMethod("transfer", list)
	fmt.Println(hex.EncodeToString(data))
	fmt.Println(hexutil.Encode(data))
	fmt.Println(err)
}
