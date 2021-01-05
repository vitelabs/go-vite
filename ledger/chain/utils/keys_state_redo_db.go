package chain_utils

import "github.com/vitelabs/go-vite/common/types"

// -------------------------------
type SnapshotKey [1 + types.HeightSize]byte

func (key SnapshotKey) Bytes() []byte {
	return key[:]
}

func (key SnapshotKey) String() string {
	return string(key[:])
}

func (key *SnapshotKey) HeightRefill(height uint64) {
	Uint64Put(key[1:1+types.HeightSize], height)
}
