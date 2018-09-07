package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/big"
)

type AccountBlockMeta struct {
	// Account id
	AccountId *big.Int

	// Height
	Height *big.Int

	// Block status, 1 means open, 2 means closed
	Status int

	// Is snapshotted
	IsSnapshot bool
}

func (*AccountBlockMeta) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*AccountBlockMeta) DbDeserialize([]byte) error {
	return nil
}

type AccountBlock struct {
	Meta *AccountBlockMeta

	BlockType byte
	Hash      *types.Hash
	Height    *big.Int
	PrevHash  *types.Hash

	AccountAddress *types.Address

	PublicKey     ed25519.PublicKey
	ToAddress     *types.Address
	FromBlockHash *types.Hash

	Amount  *big.Int
	TokenId *types.TokenTypeId

	QuotaFee    *big.Int
	ContractFee *big.Int

	SnapshotHash *types.Hash
	Data         string

	Timestamp int64
	StateHash *types.Hash
	LogHash   *types.Hash

	Nonce             []byte
	SendBlockHashList []*types.Hash
	Signature         []byte
}

func (*AccountBlock) GetComputeHash() *types.Hash {
	return nil
}

func (*AccountBlock) VerifySignature() bool {
	return true
}

func (*AccountBlock) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*AccountBlock) DbDeserialize([]byte) error {
	return nil
}

func (*AccountBlock) NetSerialize() ([]byte, error) {
	return nil, nil
}

func (*AccountBlock) NetDeserialize([]byte) error {
	return nil
}

func (*AccountBlock) FileSerialize([]byte) ([]byte, error) {
	return nil, nil
}

func (*AccountBlock) FileDeserialize([]byte) error {
	return nil
}
