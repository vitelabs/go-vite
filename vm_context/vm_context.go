package vm_context

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"math/big"
)

const (
	ACTION_ADD_BALANCE = iota
	ACTION_SUB_BALANCE
	ACTION_SET_GID
	ACTION_SET_CONTRACT
	ACTION_SET_TOKEN
	ACTION_SET_STORAGE
	ACTION_ADD_LOG
)

var (
	STORAGE_KEY_BALANCE = []byte("$balance")
	STORAGE_KEY_CODE    = []byte("$code")
)

type Action struct {
	ActionType int32
	Params     []interface{}
}

func NewAction(actionType int32, params []interface{}) *Action {
	return &Action{
		ActionType: actionType,
		Params:     params,
	}
}

type VmContext struct {
	chain   Chain
	address *types.Address

	currentSnapshotBlock *ledger.SnapshotBlock
	prevAccountBlock     *ledger.AccountBlock
	trie                 *trie.Trie

	actionList []*Action
	cache      *VmContextCache
}

func NewVmContext(chain Chain, snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) (*VmContext, error) {
	vmContext := &VmContext{
		chain:   chain,
		address: addr,

		cache: NewVmContextCache(),
	}

	currentSnapshotBlock, err := chain.GetSnapshotBlockByHash(snapshotBlockHash)
	if err != nil {
		return nil, err
	}

	prevAccountBlock, err := chain.GetAccountBlockByHash(prevAccountBlockHash)
	if err != nil {
		return nil, err
	}

	vmContext.currentSnapshotBlock = currentSnapshotBlock
	vmContext.prevAccountBlock = prevAccountBlock
	vmContext.trie = chain.GetStateTrie(prevAccountBlockHash)

	return vmContext, nil
}

func (context *VmContext) addAction(actionType int32, params []interface{}) {
	context.actionList = append(context.actionList, NewAction(actionType, params))
}

func (context *VmContext) Address() *types.Address {
	return context.address
}

func (context *VmContext) ActionList() []*Action {
	return context.actionList
}

func (context *VmContext) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	var balance = big.NewInt(0)
	if bytes.Equal(addr.Bytes(), context.Address().Bytes()) {
		if cacheBalance := context.cache.balance[*tokenTypeId]; cacheBalance != nil {
			balance = cacheBalance
		} else {
			balanceBytes := context.trie.GetValue(append(STORAGE_KEY_BALANCE, tokenTypeId.Bytes()...))
			if balanceBytes != nil {
				balance.SetBytes(balanceBytes)
			}
		}
	} else {
		latestAccountBlock := context.GetLatestAccountBlock(addr)
		if latestAccountBlock != nil {
			trie := context.chain.GetStateTrie(&latestAccountBlock.StateHash)
			balanceBytes := trie.GetValue(append(STORAGE_KEY_BALANCE, tokenTypeId.Bytes()...))
			if balanceBytes != nil {
				balance.SetBytes(balanceBytes)
			}
		}
	}

	return balance
}

// TODO: 当账号不存在时创建账号
func (context *VmContext) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	var currentBalance = big.NewInt(0)
	if cacheBalance := context.cache.balance[*tokenTypeId]; cacheBalance != nil {
		currentBalance = cacheBalance
	}
	currentBalance.Add(currentBalance, amount)
	context.cache.balance[*tokenTypeId] = currentBalance

	context.addAction(ACTION_ADD_BALANCE, []interface{}{tokenTypeId, amount})
}

func (context *VmContext) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	currentBalance := context.cache.balance[*tokenTypeId]
	if currentBalance == nil {
		return
	}
	currentBalance.Sub(currentBalance, amount)
	if currentBalance.Cmp(big.NewInt(0)) < 0 {
		return
	}

	context.cache.balance[*tokenTypeId] = currentBalance

	context.addAction(ACTION_SUB_BALANCE, []interface{}{tokenTypeId, amount})
}

func (context *VmContext) GetSnapshotBlock(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (context *VmContext) GetSnapshotBlocks(startHeight uint64, count uint64, forward bool) []*ledger.SnapshotBlock {
	return nil
}

func (context *VmContext) GetSnapshotBlockByHeight(height uint64) *ledger.SnapshotBlock {

	return nil
}

func (context *VmContext) Reset() {

}

func (context *VmContext) SetContractGid(gid *types.Gid, addr *types.Address, open bool) {

}

func (context *VmContext) SetContractCode(gid *types.Gid, code []byte) {

}

func (context *VmContext) GetContractCode(addr *types.Address) []byte {
	return nil
}

func (context *VmContext) SetToken(token *ledger.Token) {

}

func (context *VmContext) GetToken(id *types.TokenTypeId) *ledger.Token {
	return nil
}

func (context *VmContext) SetStorage(key []byte, value []byte) {

}

func (context *VmContext) GetStorage(addr *types.Address, key []byte) []byte {
	return nil
}

func (context *VmContext) GetStorageHash() *types.Hash {
	return nil
}

func (context *VmContext) GetGid() *types.Gid {
	return nil
}

func (context *VmContext) AddLog(log *ledger.VmLog) {
	context.addAction(ACTION_ADD_LOG, []interface{}{log})
	context.cache.logList = append(context.cache.logList, log)
}

func (context *VmContext) GetLogListHash() *types.Hash {
	return context.cache.logList.Hash()
}

func (context *VmContext) IsAddressExisted(addr *types.Address) bool {
	account := context.chain.GetAccount(addr)
	if account == nil {
		return false
	}
	return true
}

func (context *VmContext) GetLatestAccountBlock(addr *types.Address) *ledger.AccountBlock {
	return nil
}

func (context *VmContext) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	accountBlock, _ := context.chain.GetAccountBlockByHash(hash)
	return accountBlock
}

func (context *VmContext) NewStorageIterator(prefix []byte) *StorageIterator {
	return &StorageIterator{}
}
