package api

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
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
	prevHash, err := types.HexToHash("")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("zero-tkId %v zero-addr %v\n", zeroTkId, zeroAddr)

	addr, err := types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
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

	privkey, err := ed25519.HexToPrivateKey("")
	if err != nil {
		t.Fatal(err)
	}
	pubKey := privkey.PubByte()

	address := types.PubkeyToAddress(pubKey)
	if address != addr {
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

	signData := ed25519.Sign(privkey, hashData.Bytes())
	signBase64 := base64.StdEncoding.EncodeToString(signData)
	pubKeyBase64 := base64.StdEncoding.EncodeToString(pubKey)

	fmt.Printf("sig=%v\n pub=%v\n hash=%v\n", signBase64, pubKeyBase64, hashData)

	/*	if block.ComputeHash() != hash {
			t.Fatal("compute hash failed")
		}

		isVerified, verifyErr := crypto.VerifySig(pubSlice, hash.Bytes(), sigSlice)
		if verifyErr != nil {
			t.Fatal(verifyErr)
		}
		if !isVerified {
			t.Fatal("verify hash failed")
		}*/
}

func TestTx_Auto(t *testing.T) {

}
