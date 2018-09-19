package ledger

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"time"
)

type SnapshotContent map[types.Address]*SnapshotContentItem

func (*SnapshotContent) Serialize() ([]byte, error) {
	return nil, nil
}
func (*SnapshotContent) Deserialize([]byte) error {
	return nil
}

func (*SnapshotContent) Hash() []byte {
	return nil
}

type SnapshotContentItem struct {
	AccountBlockHeight uint64
	AccountBlockHash   types.Hash
}

type SnapshotBlock struct {
	Hash     types.Hash
	PrevHash types.Hash
	Height   uint64
	producer *types.Address

	PublicKey ed25519.PublicKey
	Signature []byte

	Timestamp *time.Time

	SnapshotHash    types.Hash
	SnapshotContent SnapshotContent
}

func (sb *SnapshotBlock) ComputeHash() types.Hash {
	var source []byte
	// PrevHash
	source = append(source, sb.PrevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, sb.Height)
	source = append(source, heightBytes...)

	// PublicKey
	source = append(source, sb.PublicKey...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(sb.Timestamp.Unix()))
	source = append(source, unixTimeBytes...)

	// SnapshotHash
	source = append(source, sb.SnapshotHash.Bytes()...)

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return hash
}

func (sb *SnapshotBlock) Producer() types.Address {
	if sb.producer == nil {
		producer := types.PubkeyToAddress(sb.PublicKey)
		sb.producer = &producer
	}
	return *sb.producer
}

func (sb *SnapshotBlock) VerifySignature() bool {
	isVerified, verifyErr := crypto.VerifySig(sb.PublicKey, sb.Hash.Bytes(), sb.Signature)
	if verifyErr != nil {
		accountBlockLog.Error("crypto.VerifySig failed, error is "+verifyErr.Error(), "method", "VerifySignature")
	}
	return isVerified
}

func (*SnapshotBlock) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*SnapshotBlock) DbDeserialize([]byte) error {
	return nil
}

func (*SnapshotBlock) Serialize() ([]byte, error) {
	return nil, nil
}

func (*SnapshotBlock) Deserialize([]byte) error {
	return nil
}

func GetGenesesSnapshotBlock() *SnapshotBlock {
	return nil
}
