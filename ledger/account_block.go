package ledger

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"time"
)

var accountBlockLog = log15.New("module", "ledger/account_block")

type AccountBlockMeta struct {
	// Account id
	AccountId uint64

	// Height
	Height uint64

	// Block status, 1 means open, 2 means closed
	ReceiveBlockHeight uint64

	// Height of Snapshot block which confirm this account block
	SnapshotHeight uint64

	// Height of Snapshot block which pointed by this account block
	RefSnapshotHeight uint64
}

func (*AccountBlockMeta) Serialize() ([]byte, error) {
	return nil, nil
}

func (*AccountBlockMeta) Deserialize([]byte) error {
	return nil
}

const (
	BlockTypeSendCreate byte = iota + 1
	BlockTypeSendCall
	BlockTypeSendReward
	BlockTypeReceive
	BlockTypeReceiveError
)

type AccountBlock struct {
	Meta *AccountBlockMeta

	BlockType byte
	Hash      types.Hash
	Height    uint64
	PrevHash  types.Hash

	AccountAddress types.Address

	PublicKey     ed25519.PublicKey
	ToAddress     types.Address
	FromBlockHash types.Hash

	Amount  *big.Int
	TokenId types.TokenTypeId

	Quota uint64
	Fee   *big.Int

	SnapshotHash types.Hash
	Data         []byte

	Timestamp *time.Time
	StateHash types.Hash

	LogHash *types.Hash

	Nonce     []byte
	Signature []byte
}

func (*AccountBlock) Copy() *AccountBlock {
	return nil
}

func (ab *AccountBlock) Proto() {

}

func (ab *AccountBlock) DeProto() {

}

func (ab *AccountBlock) ComputeHash() types.Hash {
	var source []byte
	// BlockType
	source = append(source, ab.BlockType)

	// PrevHash
	source = append(source, ab.PrevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, ab.Height)
	source = append(source, heightBytes...)

	// AccountAddress
	source = append(source, ab.AccountAddress.Bytes()...)

	if ab.IsSendBlock() {
		// ToAddress
		source = append(source, ab.ToAddress.Bytes()...)
		// Amount
		source = append(source, ab.Amount.Bytes()...)
		// TokenId
		source = append(source, ab.TokenId.Bytes()...)
	} else {
		// FromBlockHash
		source = append(source, ab.FromBlockHash.Bytes()...)
	}

	// Quota
	quotaBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(quotaBytes, ab.Quota)
	source = append(source, quotaBytes...)

	// Fee
	source = append(source, ab.Fee.Bytes()...)

	// SnapshotHash
	source = append(source, ab.SnapshotHash.Bytes()...)

	// Data
	source = append(source, ab.Data...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(ab.Timestamp.Unix()))
	source = append(source, unixTimeBytes...)

	// StateHash
	source = append(source, ab.StateHash.Bytes()...)

	// LogHash
	if ab.LogHash != nil {
		source = append(source, ab.LogHash.Bytes()...)
	}

	// Nonce
	source = append(source, ab.Nonce...)

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return hash
}

func (ab *AccountBlock) VerifySignature() bool {
	isVerified, verifyErr := crypto.VerifySig(ab.PublicKey, ab.Hash.Bytes(), ab.Signature)
	if verifyErr != nil {
		accountBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

func (ab *AccountBlock) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*AccountBlock) DbDeserialize([]byte) error {
	return nil
}

func (*AccountBlock) Serialize() ([]byte, error) {
	return nil, nil
}

func (*AccountBlock) Deserialize([]byte) error {
	return nil
}

func (ab *AccountBlock) IsSendBlock() bool {
	return ab.BlockType == BlockTypeSendCreate || ab.BlockType == BlockTypeSendCall || ab.BlockType == BlockTypeSendReward
}

func (ab *AccountBlock) IsReceiveBlock() bool {
	return ab.BlockType == BlockTypeReceive || ab.BlockType == BlockTypeReceiveError
}

func GenesesMintageBlock() *AccountBlock {
	return nil
}

func GenesesMintageReceiveBlock() *AccountBlock {
	return nil
}

func GenesesCreateGroupBlock() *AccountBlock {
	return nil
}

func GenesesCreateGroupReceiveBlock() *AccountBlock {
	return nil
}

func (ab AccountBlock) IsContractTx() bool {
	//return len(ab.Code) != 0
	return false
}