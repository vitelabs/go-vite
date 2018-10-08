package message

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

// @section GetSnapshotBlocks

type GetSnapshotBlocks struct {
	From    *ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetSnapshotBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (b *GetSnapshotBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section SnapshotBlocks

type SnapshotBlocks struct {
	Blocks []*ledger.SnapshotBlock
}

func (b *SnapshotBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (b *SnapshotBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section SubLedger

type SubLedger struct {
	SBlocks []*ledger.SnapshotBlock
	ABlocks map[types.Address][]*ledger.AccountBlock
}

func (s *SubLedger) Serialize() ([]byte, error) {
	panic("implement me")
}

func (s *SubLedger) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section GetAccountBlocks

type GetAccountBlocks struct {
	Address types.Address // maybe nil
	From    *ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetAccountBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (b *GetAccountBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section AccountBlocks

type AccountBlocks struct {
	Address types.Address
	Blocks  []*ledger.AccountBlock
}

func (a *AccountBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (a *AccountBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}
