package ledger

import (
	"math/big"

	"github.com/golang/protobuf/proto"
	"go-vite/vitepb"
)

type AccountBlock struct {
	// self account
	Account []byte

	// Receiver account, exists in send block
	To []byte

	// Last block hash
	PrevHash []byte

	// Correlative send block hash, exists in receive block
	FromHash []byte

	// Height of account chain
	BlockNum *big.Int

	// Balance of current account
	Balance map[uint32]*big.Int

	// Signature of current block
	Signature []byte
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