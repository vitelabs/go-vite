package net

import (
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type mock_verifier struct {
}

func (mv mock_verifier) VerifyNetSb(block *ledger.SnapshotBlock) error {
	return nil
}

func (mv mock_verifier) VerifyNetAb(block *ledger.AccountBlock) error {
	return nil
}

func TestVerifier_VerifyNetSb(t *testing.T) {
	wv := newVerifier(mock_verifier{})

	hash, _ := types.HexToHash("b07c664782e2e0b439191d0bba5d9a97bdbc640ab1008569bafb1b836fc0a034")
	block := &ledger.SnapshotBlock{
		Hash: hash,
	}

	err := wv.VerifyNetSb(block)
	if err == nil {
		t.Fatalf("should be blocked")
	}

	block.Hash = types.Hash{}
	err = wv.VerifyNetSb(block)
	if err != nil {
		t.Fatalf("err should be nil")
	}
}
