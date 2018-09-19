package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type Signer interface {
	SignData(a types.Address, data []byte) (signedData, pubkey []byte, err error)
	SignDataWithPassphrase(a types.Address, passphrase string, data []byte) (signedData, pubkey []byte, err error)
}

type Chain interface {
	vm_context.Chain
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
}
