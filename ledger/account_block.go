package ledger

import (
	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
)

type AccountBlockMeta struct {
	// Account id
	AccountId *big.Int

	// AccountBlock height
	Height *big.Int

	// AccountBlock status
	Status int
}


type AccountBlock struct {
	// AccountBlock height
	Height *big.Int

	// Self account
	AccountAddress []byte

	// Receiver account, exists in send block
	To []byte

	// Correlative send block hash, exists in receive block
	FromHash []byte

	// Last block hash
	PrevHash []byte

	// Block hash
	Hash []byte

	// Balance of current account
	Balance *big.Int

	// Amount of this transaction
	Amount *big.Int

	// Id of token received or sent
	TokenId []byte

	// Height of last transaction block in this token
	LastBlockHeightInToken *big.Int

	// Data requested or repsonsed
	Data string

	// Snapshot timestamp
	SnapshotTimestamp []byte

	// Signature of current block
	Signature []byte

	// PoW nounce
	Nounce []byte

	// PoW difficulty
	Difficulty []byte

	// Service fee
	FAmount *big.Int
}

func (ab *AccountBlock) DbSerialize () ([]byte, error) {
	accountBlockPB := &vitepb.AccountBlockDb{
		To: ab.To,

		PrevHash: ab.PrevHash,
		FromHash: ab.FromHash,

		TokenId: ab.TokenId,

		Balance: ab.Balance.Bytes(),

		Data: ab.Data,
		SnapshotTimestamp: ab.SnapshotTimestamp,

		Signature: ab.Signature,

		Nounce: ab.Nounce,
		Difficulty: ab.Difficulty,

		FAmount: ab.FAmount.Bytes(),
	}

	serializedBytes, err := proto.Marshal(accountBlockPB)

	if err != nil {
		return nil, err
	}

	return serializedBytes, nil
}
//
//
//func (ab *AccountBlock) Deserialize (buf []byte) error {
//	accountBlockPB := &vitepb.AccountBlock{}
//	if err := proto.Unmarshal(buf, accountBlockPB); err != nil {
//		return err
//	}
//
//	ab.Account = accountBlockPB.Account
//	ab.To = accountBlockPB.To
//
//	ab.PrevHash = accountBlockPB.PrevHash
//	ab.FromHash = accountBlockPB.FromHash
//
//	ab.BlockNum = &big.Int{}
//	ab.BlockNum.SetBytes(accountBlockPB.BlockNum)
//
//	ab.setBalanceByBytes(accountBlockPB.Balance)
//
//	ab.Signature = accountBlockPB.Signature
//
//	return nil
//}