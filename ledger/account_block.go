package ledger

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
)

var accountBlockLog = log15.New("module", "ledger/account_block")

type AccountBlockMeta struct {
	// Account id
	AccountId uint64 `json:"accountId"`

	// Height
	Height uint64 `json:"height"`

	// Block status, 1 means open, 2 means closed
	ReceiveBlockHeights []uint64 `json:"receiveBlockHeights"`

	// Height of Snapshot block which confirm this account block
	SnapshotHeight uint64 `json:"snapshotHeight"`

	// Height of Snapshot block which pointed by this account block
	RefSnapshotHeight uint64 `json:"refSnapshotHeight"`
}

func (abm *AccountBlockMeta) Copy() *AccountBlockMeta {
	newAbm := *abm

	newAbm.ReceiveBlockHeights = make([]uint64, len(abm.ReceiveBlockHeights))
	copy(newAbm.ReceiveBlockHeights, abm.ReceiveBlockHeights)

	return &newAbm
}

func (abm *AccountBlockMeta) Proto() *vitepb.AccountBlockMeta {
	pb := &vitepb.AccountBlockMeta{
		AccountId:           abm.AccountId,
		Height:              abm.Height,
		ReceiveBlockHeights: abm.ReceiveBlockHeights,

		RefSnapshotHeight: abm.RefSnapshotHeight,
	}
	return pb
}

func (abm *AccountBlockMeta) DeProto(pb *vitepb.AccountBlockMeta) {
	abm.AccountId = pb.AccountId
	abm.Height = pb.Height
	abm.ReceiveBlockHeights = pb.ReceiveBlockHeights
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
	BlockTypeSendRefund
)

type AccountBlock struct {
	Meta *AccountBlockMeta `json:"-"`

	producer *types.Address

	BlockType byte       `json:"blockType"`
	Hash      types.Hash `json:"hash"`
	Height    uint64     `json:"height"`
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`

	PublicKey     ed25519.PublicKey `json:"publicKey"`
	ToAddress     types.Address     `json:"toAddress"`
	FromBlockHash types.Hash        `json:"fromBlockHash"`

	Amount  *big.Int          `json:"amount"`
	TokenId types.TokenTypeId `json:"tokenId"`

	Quota uint64   `json:"quota"`
	Fee   *big.Int `json:"fee"`

	SnapshotHash types.Hash `json:"snapshotHash"`
	Data         []byte     `json:"data"`

	Timestamp *time.Time `json:"timestamp"`
	StateHash types.Hash `json:"stateHash"`

	LogHash *types.Hash `json:"logHash"`

	Difficulty    *big.Int        `json:"difficulty"`
	Nonce         []byte          `json:"nonce"`
	SendBlockList []*AccountBlock `json:sendBlockList` // 13
	Signature     []byte          `json:"signature"`
}

func (ab *AccountBlock) Copy() *AccountBlock {
	newAb := *ab

	if ab.Meta != nil {
		newAb.Meta = ab.Meta.Copy()
	}

	if ab.producer != nil {
		producer := *ab.producer
		newAb.producer = &producer
	}

	if ab.Amount != nil {
		newAb.Amount = new(big.Int).Set(ab.Amount)
	}

	if ab.Fee != nil {
		newAb.Fee = new(big.Int).Set(ab.Fee)
	}
	newAb.SnapshotHash = ab.SnapshotHash

	newAb.Data = make([]byte, len(ab.Data))
	copy(newAb.Data, ab.Data)

	// TODO
	if ab.Timestamp != nil {
		timestamp := time.Unix(0, ab.Timestamp.UnixNano())
		newAb.Timestamp = &timestamp
	} else {
		t := time.Now()
		newAb.Timestamp = &t
	}

	if ab.LogHash != nil {
		logHash := *ab.LogHash
		newAb.LogHash = &logHash
	}

	if ab.Difficulty != nil {
		newAb.Difficulty = new(big.Int).Set(ab.Difficulty)
	}

	newAb.Nonce = make([]byte, len(ab.Nonce))
	copy(newAb.Nonce, ab.Nonce)

	newAb.Signature = make([]byte, len(ab.Signature))
	copy(newAb.Signature, ab.Signature)
	return &newAb
}

func (ab *AccountBlock) Producer() types.Address {
	if ab.producer == nil {
		if len(ab.PublicKey) > 0 {
			producer := types.PubkeyToAddress(ab.PublicKey)
			ab.producer = &producer
		} else {
			ab.producer = &ab.AccountAddress
		}

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

	fee := ab.Fee
	if fee == nil {
		fee = big.NewInt(0)
	}
	pb.Fee = fee.Bytes()

	pb.SnapshotHash = ab.SnapshotHash.Bytes()
	pb.Data = ab.Data
	pb.Timestamp = ab.Timestamp.UnixNano()

	if ab.LogHash != nil {
		pb.LogHash = ab.LogHash.Bytes()
	}

	if ab.Difficulty != nil {
		pb.Difficulty = ab.Difficulty.Bytes()
		// Difficulty is big.NewInt(0)
		if len(pb.Difficulty) <= 0 {
			pb.Difficulty = []byte{0}
		}
	}
	pb.Nonce = ab.Nonce
	pb.Signature = ab.Signature
	return pb
}

func (ab *AccountBlock) DbProto() *vitepb.AccountBlock {
	pb := ab.proto()
	pb.StateHash = ab.StateHash.Bytes()
	if len(ab.Producer()) > 0 && !bytes.Equal(ab.Producer().Bytes(), ab.AccountAddress.Bytes()) {
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

	if len(pb.ToAddress) > 0 {
		ab.ToAddress, _ = types.BytesToAddress(pb.ToAddress)
	}
	if len(pb.TokenId) > 0 {
		ab.TokenId, _ = types.BytesToTokenTypeId(pb.TokenId)
	}

	ab.Amount = big.NewInt(0)

	if len(pb.Amount) > 0 {
		ab.Amount.SetBytes(pb.Amount)
	}

	if len(pb.FromBlockHash) > 0 {
		ab.FromBlockHash, _ = types.BytesToHash(pb.FromBlockHash)
	}
	ab.Quota = pb.Quota

	ab.Fee = big.NewInt(0)
	if len(pb.Fee) > 0 {
		ab.Fee.SetBytes(pb.Fee)
	}

	ab.SnapshotHash, _ = types.BytesToHash(pb.SnapshotHash)
	ab.Data = pb.Data
	timestamp := time.Unix(0, pb.Timestamp)
	ab.Timestamp = &timestamp
	ab.StateHash, _ = types.BytesToHash(pb.StateHash)

	if len(pb.LogHash) > 0 {
		logHash, _ := types.BytesToHash(pb.LogHash)
		ab.LogHash = &logHash
	}

	if len(pb.Difficulty) > 0 {
		ab.Difficulty = new(big.Int).SetBytes(pb.Difficulty)
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

	// Fee
	fee := ab.Fee
	if fee == nil {
		fee = big.NewInt(0)
	}
	source = append(source, fee.Bytes()...)

	// SnapshotHash
	source = append(source, ab.SnapshotHash.Bytes()...)

	// Data
	source = append(source, ab.Data...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(ab.Timestamp.Unix()))
	source = append(source, unixTimeBytes...)

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
	return ab.BlockType == BlockTypeSendCreate ||
		ab.BlockType == BlockTypeSendCall ||
		ab.BlockType == BlockTypeSendReward ||
		ab.BlockType == BlockTypeSendRefund
}

func (ab *AccountBlock) IsReceiveBlock() bool {
	return ab.BlockType == BlockTypeReceive || ab.BlockType == BlockTypeReceiveError
}
