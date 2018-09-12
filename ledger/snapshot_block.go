package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"time"
)

type SnapshotContentItem struct {
	AccountBlockHeight uint64
	AccountBlockHash   types.Hash
}

type SnapshotBlock struct {
	Hash     types.Hash
	PrevHash types.Hash
	Height   uint64
	Producer types.Address

	PublicKey ed25519.PublicKey
	Signature []byte

	Timestamp *time.Time

	SnapshotHash    types.Hash
	SnapshotContent map[types.Address]*SnapshotContentItem
}

func (*SnapshotBlock) ComputeHash() types.Hash {
	hash, _ := types.BytesToHash([]byte("abcdeabcdeabcdeabcde"))
	return hash
}

func (*SnapshotBlock) VerifySignature() bool {
	return true
}

func (*SnapshotBlock) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*SnapshotBlock) DbDeserialize([]byte) error {
	return nil
}

func (*SnapshotBlock) NetSerialize() ([]byte, error) {
	return nil, nil
}

func (*SnapshotBlock) NetDeserialize([]byte) error {
	return nil
}

func (*SnapshotBlock) FileSerialize([]byte) ([]byte, error) {
	return nil, nil
}

func (*SnapshotBlock) FileDeserialize([]byte) error {
	return nil
}

func GetGenesesSnapshotBlock() *SnapshotBlock {
	return nil
}
