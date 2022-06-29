package api

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/pow"
)

func TestPubKeyToAddress(t *testing.T) {
	publicKey, err := base64.StdEncoding.DecodeString("")
	if err != nil {
		t.Fatal(err)
	}
	addr := types.PubkeyToAddress(publicKey)
	fmt.Printf("publicKey to addr %v\n", addr)
}

func TestTx_SendRawTx_VerifyHashAndSig(t *testing.T) {
	/*	hash, err := types.HexToHash("")
		if err != nil {
			t.Fatal(err)
		}
		sigSlice, err := base64.StdEncoding.DecodeString("")
		if err != nil {
			t.Fatal(err)
		}
		publicKey, err := base64.StdEncoding.DecodeString("")
		if err != nil {
			t.Fatal(err)
		}

		data, err := base64.StdEncoding.DecodeString("5L2g5aW9")
		if err != nil {
			t.Fatal(err)
		}

		fromBlockHash, err := types.HexToHash("")
		if err != nil {
			t.Fatal(err)
		}*/

	zeroAddr, err := types.BytesToAddress(types.ZERO_ADDRESS.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	zeroTkId, err := types.BytesToTokenTypeId(types.ZERO_TOKENID.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	tokenId, err := types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	if err != nil {
		t.Fatal(err)
	}
	prevHash, err := types.HexToHash("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("zero-tkId %v zero-addr %v\n", zeroTkId, zeroAddr)

	addr, err := types.HexToAddress("vite_6c1032417f80329f3abe0a024fa3a7aa0e952b0fded2262f6f")
	if err != nil {
		t.Fatal(err)
	}

	difficulty := "65535"
	powDataHash := types.DataHash(append(addr.Bytes(), prevHash.Bytes()...))
	fmt.Printf("pow difficulty %v dataHash %v\n", difficulty, powDataHash)
	nonce, err := base64.StdEncoding.DecodeString("dfBL1GFpMNA=")
	if err != nil {
		t.Fatal(err)
	}

	privkey, err := ed25519.HexToPrivateKey("44e9768b7d8320a282e75337df8fc1f12a4f000b9f9906ddb886c6823bb599addfda7318e7824d25aae3c749c1cbd4e72ce9401653c66479554a05a2e3cb4f88")
	if err != nil {
		t.Fatal(err)
	}
	pubKey := privkey.PubByte()

	address := types.PubkeyToAddress(pubKey)
	if address != addr {
		fmt.Printf("expected address: %s / actual address: %s\n", addr, address)
		t.Fatal("publicKey doesn't match address")
	}

	amount := big.NewInt(10000)
	block := &ledger.AccountBlock{
		BlockType:      2,
		PrevHash:       prevHash,
		Height:         2,
		AccountAddress: addr,
		PublicKey:      pubKey,
		ToAddress:      addr,
		Amount:         amount,
		TokenId:        tokenId,
		/*		FromBlockHash:  fromBlockHash*/
		/*		Data: data,*/
		Difficulty: big.NewInt(65535),
		Nonce:      nonce,
	}

	hashData := block.ComputeHash()
	if hashData.IsZero() {
		t.Fatal("compute hash failed")
	}

	signData := ed25519.Sign(privkey, hashData.Bytes())
	signBase64 := base64.StdEncoding.EncodeToString(signData)
	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubKey)

	fmt.Printf("sig=%v\n pub=%v\n hash=%v\n", signBase64, pubKeyBase64, hashData)

	isVerified, verifyErr := crypto.VerifySig(pubKey, hashData.Bytes(), signData)
	if verifyErr != nil {
		t.Fatal(verifyErr)
	}
	if !isVerified {
		t.Fatal("verify hash failed")
	}
}

func TestTx_Auto(t *testing.T) {

}

func TestPow(t *testing.T) {
	address := types.HexToAddressPanic("vite_f1a9bed77ce7caf9774d0bb82b98e0946570b3531f8f554a00")
	hash := types.HexToHashPanic("92a44a90ca60b4bdf4dbdcff5f3452df892271c217e492910013f5c6be6e22ec")

	//go func() {
	//	listenAddress := fmt.Sprintf("%s:%d", "0.0.0.0", 8009)
	//	http.ListenAndServe(listenAddress, nil)
	//}()
	now := time.Now()
	dataHash := types.DataHash(append(address.Bytes(), hash.Bytes()...))
	difficulty := "12108863"
	realDifficulty, _ := new(big.Int).SetString(difficulty, 10)
	i := uint64(0)
	step := uint64(1000000)
	for {
		s := time.Now()
		nonce, _, _ := pow.MapPowNonce2(realDifficulty, dataHash, step)
		if nonce != nil {
			check := pow.CheckPowNonce(realDifficulty, nonce, dataHash.Bytes())
			t.Log(base64.StdEncoding.EncodeToString(nonce), time.Now().Sub(now).String(), check)
			break
		}
		fmt.Println(step, time.Now().Sub(s).String())
		i = i + step
	}
	// tx_test.go:149: AAAAAAX1IF0= 99950685 950685 37.093220171s

}

func Benchmark_Rand(b *testing.B) {
	for i := 0; i < b.N; i++ { //use b.N for looping
		crypto.GetEntropyCSPRNG(8)
	}
}
