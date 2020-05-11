package face

import "github.com/vitelabs/go-vite/interval/common"

type ChainReader interface {
	SnapshotReader
	AccountReader
}
type SnapshotReader interface {
	GenesisSnapshot() (*common.SnapshotBlock, error)
	HeadSnapshot() (*common.SnapshotBlock, error)
	GetSnapshotByHashH(hashH common.HashHeight) *common.SnapshotBlock
	GetSnapshotByHash(hash common.Hash) *common.SnapshotBlock
	GetSnapshotByHeight(height common.Height) *common.SnapshotBlock
	//ListSnapshotBlock(limit int) []*common.SnapshotBlock
}
type AccountReader interface {
	HeadAccount(address common.Address) (*common.AccountStateBlock, error)
	GetAccountByHashH(address common.Address, hashH common.HashHeight) *common.AccountStateBlock
	GetAccountByHash(address common.Address, hash common.Hash) *common.AccountStateBlock
	GetAccountByHeight(address common.Address, height common.Height) *common.AccountStateBlock
	//ListAccountBlock(address string, limit int) []*common.AccountStateBlock

	GetAccountByFromHash(address common.Address, source common.Hash) *common.AccountStateBlock
	NextAccountSnapshot() (common.HashHeight, []*common.AccountHashH, error)
}

type SnapshotWriter interface {
	InsertSnapshotBlock(block *common.SnapshotBlock) error
	RollbackSnapshotBlockTo(height common.Height) ([]*common.SnapshotBlock, map[common.Address][]*common.AccountStateBlock, error)
}
type AccountWriter interface {
	InsertAccountBlock(address common.Address, block *common.AccountStateBlock) error
	RollbackAccountBlockTo(address common.Address, height common.Height) ([]*common.AccountStateBlock, error)
}

type ChainListener interface {
	SnapshotInsertCallback(block *common.SnapshotBlock)
	SnapshotRemoveCallback(block *common.SnapshotBlock)
	AccountInsertCallback(address common.Address, block *common.AccountStateBlock)
	AccountRemoveCallback(address common.Address, block *common.AccountStateBlock)
}

type SyncStatus interface {
	Done() bool
}
