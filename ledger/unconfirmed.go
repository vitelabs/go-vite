package ledger

import (
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb/proto"
)

type UnconfirmedByToken struct {
	TokenId *types.TokenTypeId
	TotalBalance *big.Int
	HashList []*types.Hash
}

type UnconfirmedMeta struct{
	AccountId *big.Int
	TotalNumber *big.Int
	UnconfirmedList []*UnconfirmedByToken
}

type UnconfirmedBlock struct {
	// Self account
	AccountAddress *types.Address

	// Receiver account, exists in send block
	To *types.Address

	// [Optional] Sender account, exists in receive block
	From *types.Address

	// Correlative send block hash, exists in receive block
	FromHash *types.Hash

	// Last block hash
	PrevHash *types.Hash

	// Block hash
	Hash *types.Hash

	// Balance of current account
	Balance *big.Int

	// Amount of this transaction
	Amount *big.Int

	// Timestamp
	Timestamp uint64

	// Id of token received or sent
	TokenId *types.TokenTypeId

	// [Optional] Height of last transaction block in this token
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

}

func (ucfb *UnconfirmedBlock) DbDeserialize (buf []byte) error {
	unconfirmedBlockPB := &vitepb.UnconfirmedBlockDb{}

	if err := proto.Unmarshal(buf, unconfirmedBlockPB); err != nil {
		return err
	}

	if unconfirmedBlockPB.To != nil {
		toAddress, err := types.BytesToAddress(unconfirmedBlockPB.To)
		if err != nil {
			return err
		}

		ucfb.To = &toAddress
	}

	if unconfirmedBlockPB.Hash != nil {
		hash, err := types.BytesToHash(unconfirmedBlockPB.Hash)
		if err != nil {
			return err
		}
		ucfb.Hash = &hash
	}

	if unconfirmedBlockPB.PrevHash != nil {
		prevHash, err := types.BytesToHash(unconfirmedBlockPB.PrevHash)
		if err != nil {
			return err
		}
		ucfb.PrevHash = &prevHash
	}

	if unconfirmedBlockPB.FromHash != nil {
		fromHash, err := types.BytesToHash(unconfirmedBlockPB.FromHash)
		if err != nil {
			return err
		}
		ucfb.FromHash = &fromHash
	}

	if unconfirmedBlockPB.TokenId != nil {
		tokenId, err := types.BytesToTokenTypeId(unconfirmedBlockPB.TokenId)
		if err != nil {
			return err
		}

		ucfb.TokenId = &tokenId
	}

	if unconfirmedBlockPB.Amount != nil {
		ucfb.Amount = &big.Int{}
		ucfb.Amount.SetBytes(unconfirmedBlockPB.Amount)
	}

	if unconfirmedBlockPB.Balance != nil {
		ucfb.Balance = &big.Int{}
		ucfb.Balance.SetBytes(unconfirmedBlockPB.Balance)
	}

	ucfb.Timestamp =  unconfirmedBlockPB.Timestamp

	ucfb.Data = unconfirmedBlockPB.Data

	ucfb.SnapshotTimestamp = unconfirmedBlockPB.SnapshotTimestamp

	ucfb.Signature = unconfirmedBlockPB.Signature

	ucfb.Nounce = unconfirmedBlockPB.Nounce

	ucfb.Difficulty = unconfirmedBlockPB.Difficulty

	return nil
}

func (ucfb *UnconfirmedBlock) DbSerialize () ([]byte, error) {
	accountBlockPB := &vitepb.AccountBlockDb{
		Timestamp: ucfb.Timestamp,
		Data: ucfb.Data,

		Signature: ucfb.Signature,

		Nounce: ucfb.Nounce,
		Difficulty: ucfb.Difficulty,
	}
	if ucfb.Hash != nil {
		accountBlockPB.Hash = ucfb.Hash.Bytes()
	}
	if ucfb.PrevHash != nil {
		accountBlockPB.Hash = ucfb.PrevHash.Bytes()
	}
	if ucfb.FromHash != nil {
		accountBlockPB.Hash = ucfb.FromHash.Bytes()
	}
	if ucfb.Amount != nil {
		accountBlockPB.Amount = ucfb.Amount.Bytes()
	}
	if ucfb.To != nil {
		accountBlockPB.To = ucfb.To.Bytes()
	}
	if ucfb.TokenId != nil {
		accountBlockPB.TokenId = ucfb.TokenId.Bytes()
	}
	if ucfb.SnapshotTimestamp != nil {
		accountBlockPB.SnapshotTimestamp = ucfb.SnapshotTimestamp
	}
	if ucfb.Balance != nil {
		accountBlockPB.Balance = ucfb.Balance.Bytes()
	}

	return proto.Marshal(accountBlockPB)
}

func (ucfm *UnconfirmedMeta) DbDeserialize (buf []byte) error {
	unconfirmedMetaPB := &vitepb.UnconfirmedMeta{}
	if err := proto.Unmarshal(buf, unconfirmedMetaPB); err != nil {
		return err
	}
	ucfm.AccountId = &big.Int{}
	ucfm.AccountId.SetBytes(unconfirmedMetaPB.AccountId)

	ucfm.TotalNumber = &big.Int{}
	ucfm.TotalNumber.SetBytes(unconfirmedMetaPB.TotalNumber)

	var unconfirmedList []*UnconfirmedByToken
	if unconfirmedMetaPB.UnconfirmedList != nil {
		for _, unconfirmedByToken := range unconfirmedMetaPB.UnconfirmedList {
			//UnconfirmedByToken
			var oneUnconfirmed = &UnconfirmedByToken{}
			if err := oneUnconfirmed.GetUnconfirmedByTokenByDbPB(unconfirmedByToken); err != nil {
				return err
			}
			unconfirmedList = append(unconfirmedList, oneUnconfirmed)
		}
		ucfm.UnconfirmedList = unconfirmedList
	}
	return nil
}

func (ucfm *UnconfirmedMeta) DbSerialize () ([]byte, error) {
	var unconfirmedList []*vitepb.UnconfirmedByToken

	for _, unconfirmedByToken := range ucfm.UnconfirmedList {
		unconfirmedByTokenPB, err := unconfirmedByToken.SetUnconfirmedByTokenIntoPB()
		if err != nil {
			return nil, err
		}
		unconfirmedList = append(unconfirmedList, unconfirmedByTokenPB)
	}

	unconfirmedMeta := &vitepb.UnconfirmedMeta{
		AccountId:ucfm.AccountId.Bytes(),
		TotalNumber:ucfm.TotalNumber.Bytes(),
		UnconfirmedList: unconfirmedList,
	}

	data, err := proto.Marshal(unconfirmedMeta)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ucfbt *UnconfirmedByToken) SetUnconfirmedByTokenIntoPB () (*vitepb.UnconfirmedByToken, error) {
	var hashList [][]byte
	for _, hash := range ucfbt.HashList {
		hashList =  append(hashList, hash.Bytes())
	}
	unconfirmedByTokenPB := &vitepb.UnconfirmedByToken{
		TokenId: ucfbt.TokenId.Bytes(),
		TotalBalance: ucfbt.TotalBalance.Bytes(),
		HashList: hashList,
	}
	return unconfirmedByTokenPB, nil
}

func (ucfbt *UnconfirmedByToken) GetUnconfirmedByTokenByDbPB (unconfirmedByTokenPB *vitepb.UnconfirmedByToken) error {
	ucfbt.TotalBalance = &big.Int{}
	ucfbt.TotalBalance.SetBytes(unconfirmedByTokenPB.TotalBalance)

	tokenId, err := types.BytesToTokenTypeId(unconfirmedByTokenPB.TokenId)
	if err != nil {
		return err
	}
	ucfbt.TokenId = &tokenId

	var hashList []*types.Hash
	for _, hashPB := range unconfirmedByTokenPB.HashList {
		hash, err := types.BytesToHash(hashPB)
		if err != nil {
			return err
		}
		hashList = append(hashList, &hash)
	}
	ucfbt.HashList = hashList

	return nil
}
