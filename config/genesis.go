package config

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type ForkPoint struct {
	Height uint64
	Hash   *types.Hash
}

type ForkPoints struct{}

type Genesis struct {
	GenesisAccountAddress *types.Address
	ForkPoints            *ForkPoints
	ContractStorageMap    map[string]map[string]string
	ContractLogsMap       map[string][]GenesisVmLog
	AccountBalanceMap     map[string]map[string]*big.Int
}
type GenesisVmLog struct {
	Data   string
	Topics []types.Hash
}
