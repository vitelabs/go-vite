package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"math/big"
)

type ContractGid struct {
	gid  *types.Gid
	addr *types.Address
	open bool
}

type ContractCode struct {
	gid  *types.Gid
	code []byte
}

type UnsavedCache struct {
	balance          map[types.TokenTypeId]*big.Int
	logList          ledger.VmLogList
	contractGidList  []*ContractGid
	contractCodeList []*ContractCode
	tokenList        []*ledger.Token
	trie             *trie.Trie
}

func NewUnsavedCache(oldTrie *trie.Trie) *UnsavedCache {
	return &UnsavedCache{
		balance: make(map[types.TokenTypeId]*big.Int),
		trie:    oldTrie.Copy(),
	}
}

func (cache *UnsavedCache) Balance() map[types.TokenTypeId]*big.Int {
	return cache.balance
}

func (cache *UnsavedCache) setBalance(tokenTypeId *types.TokenTypeId, balance *big.Int) {
	cache.balance[*tokenTypeId] = balance
}

func (cache *UnsavedCache) LogList() ledger.VmLogList {
	return cache.logList
}

func (cache *UnsavedCache) addLog(log *ledger.VmLog) {
	cache.logList = append(cache.logList, log)
}
