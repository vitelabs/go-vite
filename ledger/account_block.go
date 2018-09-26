package ledger

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
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
	ReceiveBlockHeights []uint64

	// Height of Snapshot block which confirm this account block
	SnapshotHeight uint64

	// Height of Snapshot block which pointed by this account block
	RefSnapshotHeight uint64
}

// TODO
func (*AccountBlockMeta) Copy() *AccountBlockMeta {
	return nil
}

func (abm *AccountBlockMeta) Proto() *vitepb.AccountBlockMeta {
	pb := &vitepb.AccountBlockMeta{
		AccountId:           abm.AccountId,
		Height:              abm.Height,
		ReceiveBlockHeights: abm.ReceiveBlockHeights,

		SnapshotHeight:    abm.SnapshotHeight,
		RefSnapshotHeight: abm.RefSnapshotHeight,
	}
	return pb
}

func (abm *AccountBlockMeta) DeProto(pb *vitepb.AccountBlockMeta) {
	abm.AccountId = pb.AccountId
	abm.Height = pb.Height
	abm.ReceiveBlockHeights = pb.ReceiveBlockHeights
	abm.SnapshotHeight = pb.SnapshotHeight
	abm.RefSnapshotHeight = pb.RefSnapshotHeight
}

func (abm *AccountBlockMeta) Serialize() ([]byte, error) {
	return proto.Marshal(abm.Proto())
}

func (abm *AccountBlockMeta) Deserialize(buf []byte) error {
	pb := &vitepb.AccountBlockMeta{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}

	abm.DeProto(pb)
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

	producer *types.Address

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

// TODO Deep copy
func (ab *AccountBlock) Copy() *AccountBlock {
	newAb := &AccountBlock{}

	return newAb
}

func (ab *AccountBlock) Producer() types.Address {
	if ab.producer == nil {
		producer := types.PubkeyToAddress(ab.PublicKey)
		ab.producer = &producer
	}
	return *ab.producer
}
func (ab *AccountBlock) proto() *vitepb.AccountBlock {
	pb := &vitepb.AccountBlock{}
	pb.BlockType = uint32(ab.BlockType)
	pb.Hash = ab.Hash.Bytes()
	pb.Height = ab.Height
	pb.PrevHash = ab.PrevHash.Bytes()

	if ab.IsSendBlock() {
		pb.ToAddress = ab.ToAddress.Bytes()
		pb.Amount = ab.Amount.Bytes()
		pb.TokenId = ab.TokenId.Bytes()
	} else {
		pb.FromBlockHash = ab.FromBlockHash.Bytes()
	}

	pb.Quota = ab.Quota
	if ab.Fee != nil {
		pb.Fee = ab.Fee.Bytes()
	}
	pb.SnapshotHash = ab.SnapshotHash.Bytes()
	pb.Data = ab.Data
	pb.Timestamp = ab.Timestamp.UnixNano()
	pb.StateHash = ab.StateHash.Bytes()
	if ab.LogHash != nil {
		pb.LogHash = ab.LogHash.Bytes()
	}
	pb.Nonce = ab.Nonce
	pb.Signature = ab.Signature
	return pb
}

func (ab *AccountBlock) DbProto() *vitepb.AccountBlock {
	pb := ab.proto()
	if !bytes.Equal(ab.Producer().Bytes(), ab.AccountAddress.Bytes()) {
		pb.PublicKey = ab.PublicKey
	}

	return pb
}

func (ab *AccountBlock) Proto() *vitepb.AccountBlock {
	pb := ab.proto()
	pb.AccountAddress = ab.AccountAddress.Bytes()
	pb.PublicKey = ab.PublicKey
	return pb
}

func (ab *AccountBlock) DeProto(pb *vitepb.AccountBlock) {
	ab.BlockType = byte(pb.BlockType)
	ab.Hash, _ = types.BytesToHash(pb.Hash)
	ab.Height = pb.Height
	ab.PrevHash, _ = types.BytesToHash(pb.PrevHash)
	ab.AccountAddress, _ = types.BytesToAddress(pb.AccountAddress)
	ab.PublicKey = pb.PublicKey
	if len(pb.ToAddress) >= 0 {
		ab.ToAddress, _ = types.BytesToAddress(pb.ToAddress)
	}
	if len(pb.TokenId) >= 0 {
		ab.TokenId, _ = types.BytesToTokenTypeId(pb.TokenId)
	}
	if len(pb.Amount) >= 0 {
		ab.Amount = big.NewInt(0)
		ab.Amount.SetBytes(pb.Amount)
	}

	if len(pb.FromBlockHash) >= 0 {
		ab.FromBlockHash, _ = types.BytesToHash(pb.FromBlockHash)
	}
	ab.Quota = pb.Quota
	if len(pb.Fee) >= 0 {
		ab.Fee = big.NewInt(0)
		ab.Fee.SetBytes(pb.Fee)
	}

	ab.SnapshotHash, _ = types.BytesToHash(pb.SnapshotHash)
	ab.Data = pb.Data
	timestamp := time.Unix(0, pb.Timestamp)
	ab.Timestamp = &timestamp
	ab.StateHash, _ = types.BytesToHash(pb.StateHash)
	if len(pb.LogHash) >= 0 {
		logHash, _ := types.BytesToHash(pb.LogHash)
		ab.LogHash = &logHash
	}

	ab.Nonce = pb.Nonce
	ab.Signature = pb.Signature

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
	return proto.Marshal(ab.DbProto())
}

func (ab *AccountBlock) DbDeserialize(buf []byte) error {
	pb := &vitepb.AccountBlock{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	ab.DeProto(pb)
	return nil
}

func (ab *AccountBlock) Serialize() ([]byte, error) {
	return proto.Marshal(ab.Proto())
}

func (ab *AccountBlock) Deserialize(buf []byte) error {
	pb := &vitepb.AccountBlock{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	ab.DeProto(pb)

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
