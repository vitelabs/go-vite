package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	dexProto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
)

type Account struct {
	Token           types.TokenTypeId
	Available       *big.Int
	Locked          *big.Int
	VxLocked        *big.Int
	VxUnlocking     *big.Int
	CancellingStake *big.Int
}

func (account *Account) Serialize() *dexProto.Account {
	pb := &dexProto.Account{
		Token: account.Token.Bytes(),
	}
	if account.Available != nil {
		pb.Available = account.Available.Bytes()
	}
	if account.Locked != nil {
		pb.Locked = account.Locked.Bytes()
	}
	if account.VxLocked != nil {
		pb.VxLocked = account.VxLocked.Bytes()
	}
	if account.VxUnlocking != nil {
		pb.VxUnlocking = account.VxUnlocking.Bytes()
	}
	if account.CancellingStake != nil {
		pb.CancellingStake = account.CancellingStake.Bytes()
	}
	return pb
}

func (account *Account) Deserialize(pb *dexProto.Account) {
	account.Token, _ = types.BytesToTokenTypeId(pb.Token)
	if len(pb.Available) > 0 {
		account.Available = new(big.Int).SetBytes(pb.Available)
	}
	if len(pb.Locked) > 0 {
		account.Locked = new(big.Int).SetBytes(pb.Locked)
	}
	if len(pb.VxLocked) > 0 {
		account.VxLocked = new(big.Int).SetBytes(pb.VxLocked)
	}
	if len(pb.VxUnlocking) > 0 {
		account.VxUnlocking = new(big.Int).SetBytes(pb.VxUnlocking)
	}
	if len(pb.CancellingStake) > 0 {
		account.CancellingStake = new(big.Int).SetBytes(pb.CancellingStake)
	}
}
