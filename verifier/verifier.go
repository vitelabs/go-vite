package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type VerifyResult int

const (
	FAIL VerifyResult = iota
	PENDING
	SUCCESS
)

type AccountPendingTask struct {
	Addr *types.Address
	Hash *types.Hash
}

type SnapshotPendingTask struct {
	Hash *types.Hash
}

type verifier struct {
	Sv *SnapshotVerifier
	Av *AccountVerifier
}

type NetVerifier interface {
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyNetAb(block *ledger.AccountBlock) error
}

func NewNetVerifier(sv *SnapshotVerifier, av *AccountVerifier) NetVerifier {
	return &verifier{
		Sv: sv,
		Av: av,
	}
}

func (v *verifier) VerifyNetSb(block *ledger.SnapshotBlock) error {
	return v.Sv.VerifyNetSb(block)
}

func (v *verifier) VerifyNetAb(block *ledger.AccountBlock) error {
	return v.Av.VerifyNetAb(block)
}
