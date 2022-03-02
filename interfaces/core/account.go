package core

import (
	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/vitepb"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
)

type Account struct {
	AccountAddress types.Address
	AccountId      uint64
	PublicKey      ed25519.PublicKey
}

func (account *Account) Proto() *vitepb.Account {
	pb := &vitepb.Account{}
	pb.AccountId = account.AccountId
	pb.PublicKey = account.PublicKey
	return pb
}

func (account *Account) DeProto(pb *vitepb.Account) {
	account.AccountId = pb.AccountId
	account.PublicKey = pb.PublicKey
	account.AccountAddress, _ = types.BytesToAddress(pb.PublicKey)
}

func (account *Account) Serialize() ([]byte, error) {
	return proto.Marshal(account.Proto())
}

func (account *Account) Deserialize(buf []byte) error {
	pb := &vitepb.Account{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	account.DeProto(pb)
	return nil
}
