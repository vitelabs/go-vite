package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	dexProto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
)

type Account struct {
	Token     types.TokenTypeId
	Available *big.Int
	Locked    *big.Int
}

func (account *Account) Serialize() *dexProto.Account {
	pb := &dexProto.Account{
		Token: account.Token.Bytes(),
	}

	available := account.Available
	if available == nil {
		available = big.NewInt(0)
	}
	pb.Available = available.Bytes()

	Locked := account.Locked
	if Locked == nil {
		Locked = big.NewInt(0)
	}
	pb.Locked = Locked.Bytes()

	return pb
}

func (account *Account) Deserialize(pb *dexProto.Account) {
	account.Token, _ = types.BytesToTokenTypeId(pb.Token)
	account.Available = big.NewInt(0)
	if len(pb.Available) > 0 {
		account.Available.SetBytes(pb.Available)
	}
	account.Locked = big.NewInt(0)
	if len(pb.Locked) > 0 {
		account.Locked.SetBytes(pb.Locked)
	}
}
