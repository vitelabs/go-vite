package api

import (
	"encoding/base64"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func TestTx_SendRawTx_VerifyHashAndSig(t *testing.T) {
	/*	hash, err := types.HexToHash("16565a78cd861c8e51e7194f761452e17f31d9b586483fdbbfd1a5fe1083ed6d")
		if err != nil {
			t.Fatal(err)
		}
		sigSlice, err := base64.StdEncoding.DecodeString("QlTGpop8q7LrWkVbzMCnffOohDYspQHEF7tmvIu1keF5KNDbCBk7BSosDEBzNjKMLdK3w47JeJt7IQXyNUM8Bg==")
		if err != nil {
			t.Fatal(err)
		}
		publicKey, err := base64.StdEncoding.DecodeString("P8UiTllDO/9PSMg8DrTt6g5MQuppfgTN7HF9A+UNUgA=")
			if err != nil {
			t.Fatal(err)
		}
	*/

	zeroTkId, err := types.BytesToTokenTypeId(types.ZERO_TOKENID.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	zeroAddr, err := types.BytesToAddress(types.ZERO_ADDRESS.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("zero-tkId %v zero-addr %v\n", zeroTkId, zeroAddr)

	addr, err := types.HexToAddress("")
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

	tokenId, err := types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	if err != nil {
		t.Fatal(err)
	}

	prevHash, err := types.HexToHash("e8b7c2b3264b1051cc1343f2b78c11158482f7d8a09f09101f370b88d1bbea25")
	if err != nil {
		t.Fatal(err)
	}

	fromBlockHash, err := types.HexToHash("e8b7c2b3264b1051cc1343f2b78c11158482f7d8a09f09101f370b88d1bbea25")
	if err != nil {
		t.Fatal(err)
	}

	amount := big.NewInt(10000)
	block := &ledger.AccountBlock{
		BlockType:      4,
		PrevHash:       prevHash,
		Height:         15,
		AccountAddress: addr,
		PublicKey:      pubKey,
		ToAddress:      addr,
		Amount:         amount,
		TokenId:        tokenId,
		FromBlockHash:  fromBlockHash,
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
