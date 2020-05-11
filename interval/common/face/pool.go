package face

import "github.com/vitelabs/go-vite/interval/common"

type PoolWriter interface {
	AddSnapshotBlock(block *common.SnapshotBlock) error
	AddDirectSnapshotBlock(block *common.SnapshotBlock) error

	AddAccountBlock(address common.Address, block *common.AccountStateBlock) error
	AddDirectAccountBlock(address common.Address, block *common.AccountStateBlock) error
}

type PoolReader interface {
	ExistInPool(address common.Address, requestHash common.Hash) bool // request对应的response是否在current链上
}
