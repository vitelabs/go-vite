package face

import "github.com/vitelabs/go-vite/interval/common"

type PoolWriter interface {
	AddSnapshotBlock(block *common.SnapshotBlock) error
	AddDirectSnapshotBlock(block *common.SnapshotBlock) error

	AddAccountBlock(address string, block *common.AccountStateBlock) error
	AddDirectAccountBlock(address string, block *common.AccountStateBlock) error
}

type PoolReader interface {
	ExistInPool(address string, requestHash string) bool // request对应的response是否在current链上
}
