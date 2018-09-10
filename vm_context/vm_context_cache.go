package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type VmContextCache struct {
	balance map[types.TokenTypeId]*big.Int
	logList ledger.VmLogList
}

func NewVmContextCache() *VmContextCache {
	return &VmContextCache{
		balance: make(map[types.TokenTypeId]*big.Int),
	}
}

func (cache *VmContextCache) Balance() map[types.TokenTypeId]*big.Int {
	return cache.balance
}

func (cache *VmContextCache) setBalance(tokenTypeId *types.TokenTypeId, balance *big.Int) {
	cache.balance[*tokenTypeId] = balance
}

func (cache *VmContextCache) LogList() ledger.VmLogList {
	return cache.logList
}

func (cache *VmContextCache) addLog(log *ledger.VmLog) {
	cache.logList = append(cache.logList, log)
}
