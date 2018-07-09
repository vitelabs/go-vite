package ledger

import (
	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
	"github.com/vitelabs/go-vite/common/types"
)

type AccountBlockMeta struct {
	// Account id
	AccountId *big.Int

	// AccountBlock height
	Height *big.Int

	// Block status, 0 means unknow, 1 means open, 2 means closed
	Status int
}

func (ab *AccountBlockMeta) DbSerialize () ([]byte, error) {
	accountBlockMetaPb := &vitepb.AccountBlockMeta{
		AccountId: ab.AccountId.Bytes(),
		Height: ab.Height.Bytes(),
		Status: uint32(ab.Status),
	}

	return proto.Marshal(accountBlockMetaPb)
}

func (abm *AccountBlockMeta) DbDeserialize (buf []byte) (error) {
	accountBlockMetaPb := &vitepb.AccountBlockMeta{}
	if err := proto.Unmarshal(buf, accountBlockMetaPb); err != nil {
		return err
	}

	abm.AccountId = &big.Int{}
	abm.AccountId.SetBytes(accountBlockMetaPb.AccountId)

	abm.Height = &big.Int{}
	abm.Height.SetBytes(accountBlockMetaPb.Height)

	abm.Status = int(accountBlockMetaPb.Status)

	return nil
}


type AccountBlock struct {
	// [Optional] AccountBlockMeta
	Meta *AccountBlockMeta

	// Self account
	AccountAddress *types.Address

	// Receiver account, exists in send block
	To *types.Address

	// [Optional] Sender account, exists in receive block
	From *types.Address

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

	// Service fee
	FAmount *big.Int
}


func (ab *AccountBlock) DbSerialize () ([]byte, error) {
	accountBlockPB := &vitepb.AccountBlockDb{
		Hash: ab.Hash,
		PrevHash: ab.PrevHash,
		FromHash: ab.FromHash,

		Timestamp: ab.Timestamp,
		Data: ab.Data,
		SnapshotTimestamp: ab.SnapshotTimestamp,

		Signature: ab.Signature,

		Nounce: ab.Nounce,
		Difficulty: ab.Difficulty,
	}

	if ab.To != nil {
		accountBlockPB.To = ab.To.Bytes()
	}

	if ab.TokenId != nil {
		accountBlockPB.TokenId = ab.TokenId.Bytes()
	}

	if ab.Balance != nil {
		accountBlockPB.Balance = ab.Balance.Bytes()
	}

	if ab.FAmount != nil {
		accountBlockPB.FAmount = ab.FAmount.Bytes()
	}

	return proto.Marshal(accountBlockPB)
}



func (ab *AccountBlock) DbDeserialize (buf []byte) error {
	accountBlockPB := &vitepb.AccountBlockDb{}
	if err := proto.Unmarshal(buf, accountBlockPB); err != nil {
		return err
	}

	if accountBlockPB.To != nil {
		toAddress, err := types.BytesToAddress(accountBlockPB.To)
		if err != nil {
			return err
		}

		ab.To = &toAddress
	}


	ab.Hash = accountBlockPB.Hash
	ab.PrevHash = accountBlockPB.PrevHash
	ab.FromHash = accountBlockPB.FromHash

	if accountBlockPB.TokenId != nil {
		tokenId, err := types.BytesToTokenTypeId(accountBlockPB.TokenId)
		if err != nil {
			return err
		}

		ab.TokenId = &tokenId
	}


	ab.Timestamp =  accountBlockPB.Timestamp

	ab.Balance = &big.Int{}
	ab.Balance.SetBytes(accountBlockPB.Balance)

	ab.Data = accountBlockPB.Data

	ab.SnapshotTimestamp = accountBlockPB.SnapshotTimestamp

	ab.Signature = accountBlockPB.Signature

	ab.Nounce = accountBlockPB.Nounce

	ab.Difficulty = accountBlockPB.Difficulty

	ab.FAmount = &big.Int{}
	ab.FAmount.SetBytes(accountBlockPB.FAmount)

	return nil
}

func GetGenesisBlocks () ([]*AccountBlock){
	firstBlockHash := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	secondBlockHash := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	viteMintageBlock := &AccountBlock{
		AccountAddress: &GenesisAccount,
		To: 			&MintageAddress,

		SnapshotTimestamp: GenesisSnapshotBlockHash,
		Hash:           firstBlockHash,             // mock
		Data: "{" +
			"\"tokenName\": \"vite\"," +
			"\"tokenSymbol\": \"VITE\"," +
			"\"owner\":\""+ GenesisAccount.String() +"\"," +
			"\"decimals\": 18," +
			"\"totalSupply\": \"1000000000\"" +
			"}",
	}

	genesisAccountBlock := &AccountBlock{
		AccountAddress: &GenesisAccount,
		FromHash: firstBlockHash,
		PrevHash: firstBlockHash,

		SnapshotTimestamp: GenesisSnapshotBlockHash,
		Hash: secondBlockHash,
	}

	return []*AccountBlock{viteMintageBlock, genesisAccountBlock}
}

