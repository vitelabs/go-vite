package api

import (
	"encoding/base64"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

func TestTx_SendRawTx_VerifyHashAndSig(t *testing.T) {
	hash, err := types.HexToHash("16565a78cd861c8e51e7194f761452e17f31d9b586483fdbbfd1a5fe1083ed6d")
	if err != nil {
		t.Fatal(err)
	}
	addr, err := types.HexToAddress("vite_13f1f8e230f2ffa1e030e664e525033ff995d6c2bb15af4cf9")
	if err != nil {
		t.Fatal(err)
	}
	fromHash, err := types.HexToHash("20a75cc0baf4d0b6a3eef4f486825f9f00dba00ed1b4af0aad91a48895165186")
	if err != nil {
		t.Fatal(err)
	}
	sigSlice, err := base64.StdEncoding.DecodeString("QlTGpop8q7LrWkVbzMCnffOohDYspQHEF7tmvIu1keF5KNDbCBk7BSosDEBzNjKMLdK3w47JeJt7IQXyNUM8Bg==")
	if err != nil {
		t.Fatal(err)
	}
	pubSlice, err := base64.StdEncoding.DecodeString("")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("sig=%v\n pub=%v\n", sigSlice, pubSlice)

	accountBlock := AccountBlock{
		AccountBlock: &ledger.AccountBlock{
			BlockType:      4,
			Height:         1,
			AccountAddress: addr,
			FromBlockHash:  fromHash,
			PublicKey:      pubSlice,
			Signature:      sigSlice,
		},
		Height: "1",
	}
	block, err := accountBlock.LedgerAccountBlock()
	if err != nil {
		t.Fatal(err)
	}
	if block.ComputeHash() != hash {
		t.Fatal("compute hash failed")
	}

	isVerified, verifyErr := crypto.VerifySig(pubSlice, hash.Bytes(), sigSlice)
	if verifyErr != nil {
		t.Fatal(verifyErr)
	}
	if !isVerified {
		t.Fatal("verify hash failed")
	}
}
