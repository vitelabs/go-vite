package ledger

import (
	"math/big"

	"github.com/golang/protobuf/proto"
	"go-vite/vitepb"
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

	// Balance of current account
	Balance map[uint32]*big.Int

	// Number of tokens received or sent
	Amount *big.Int

	// Name of token received or sent
	TokenName string

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
	fAmount *big.Int
}

func (ab *AccountBlock) getBalanceInBytes () map[uint32][]byte{
	var balanceInBytes = map[uint32][]byte{}

	for tokenId, num := range ab.Balance {
		balanceInBytes[tokenId] = num.Bytes()
	}

	return balanceInBytes
}

func (ab *AccountBlock) setBalanceByBytes (balanceInBytes map[uint32][]byte) {
	ab.Balance = map[uint32]*big.Int{}

	for tokenId, num := range balanceInBytes {
		ab.Balance[tokenId] = &big.Int{}
		ab.Balance[tokenId].SetBytes(num)
	}
}

func (ab *AccountBlock) Serialize () ([]byte, error) {
	accountBlockPB := & vitepb.AccountBlock{
		Account: ab.Account,
		To: ab.To,

		PrevHash: ab.PrevHash,
		FromHash: ab.FromHash,

		BlockNum: ab.BlockNum.Bytes(),

		Balance: ab.getBalanceInBytes(),

		Signature: ab.Signature,
	}

	serializedBytes, err := proto.Marshal(accountBlockPB)

	if err != nil {
		return nil, err
	}

	return serializedBytes, nil
}


func (ab *AccountBlock) Deserialize (buf []byte) error {
	accountBlockPB := &vitepb.AccountBlock{}
	if err := proto.Unmarshal(buf, accountBlockPB); err != nil {
		return err
	}

	ab.Account = accountBlockPB.Account
	ab.To = accountBlockPB.To

	ab.PrevHash = accountBlockPB.PrevHash
	ab.FromHash = accountBlockPB.FromHash

	ab.BlockNum = &big.Int{}
	ab.BlockNum.SetBytes(accountBlockPB.BlockNum)

	ab.setBalanceByBytes(accountBlockPB.Balance)

	ab.Signature = accountBlockPB.Signature

	return nil
}