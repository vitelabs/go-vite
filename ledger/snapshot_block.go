package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/big"
)

type SnapshotBlock struct {
	Hash     *types.Hash
	PrevHash *types.Hash
	Height   *big.Int
	Producer *types.Address

	PublicKey ed25519.PublicKey
	Signature []byte

	Timestamp    int64
	SnapshotHash *types.Hash
}

func (*SnapshotBlock) ComputeHash() *types.Hash {
	return nil
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
