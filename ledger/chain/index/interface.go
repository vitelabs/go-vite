package chain_index

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type Chain interface {
	LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error)

	IterateContracts(iterateFunc func(addr types.Address, meta *ledger.ContractMeta, err error) bool)
}
