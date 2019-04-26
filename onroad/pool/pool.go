package onroad_pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type OnRoadPool interface {
	WriteAccountBlock(block *ledger.AccountBlock) error
	DeleteAccountBlock(block *ledger.AccountBlock) error

	GetOnRoadTotalNumByAddr(addr types.Address) (uint64, error)
	GetFrontOnRoadBlocksByAddr(addr types.Address) ([]*ledger.AccountBlock, error)

	IsFrontOnRoadOfCaller(contract types.Address, caller types.Address, hash types.Hash) (bool, error)
}

type chainReader interface {
	LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	IsContractAccount(address types.Address) (bool, error)
	IsGenesisAccountBlock(hash types.Hash) bool
}
