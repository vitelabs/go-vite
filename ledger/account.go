package ledger

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/vitepb"
)

const (
	AccountTypeError    = 0
	AccountTypeNotExist = 1
	AccountTypeGeneral  = 2
	AccountTypeContract = 3
)

var GenesisAccountAddress, _ = types.HexToAddress("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")

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
