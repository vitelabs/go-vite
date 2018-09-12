package vm_context

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"math/big"
)

var (
	STORAGE_KEY_BALANCE = []byte("$balance")
	STORAGE_KEY_CODE    = []byte("$code")
)

type VmContext struct {
	chain   Chain
	address *types.Address

	currentSnapshotBlock *ledger.SnapshotBlock
	prevAccountBlock     *ledger.AccountBlock
	trie                 *trie.Trie

	unsavedCache *UnsavedCache
}

func NewVmContext(chain Chain, snapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash, addr *types.Address) (*VmContext, error) {
	vmContext := &VmContext{
		chain:   chain,
		address: addr,
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
	vmContext.unsavedCache = NewUnsavedCache(vmContext.trie)

	return vmContext, nil
}

func (context *VmContext) Address() *types.Address {
	return context.address
}

func (context *VmContext) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	var balance = big.NewInt(0)
	if bytes.Equal(addr.Bytes(), context.Address().Bytes()) {
		if cacheBalance := context.unsavedCache.balance[*tokenTypeId]; cacheBalance != nil {
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

func (context *VmContext) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	currentBalance := context.GetBalance(context.address, tokenTypeId)
	currentBalance.Add(currentBalance, amount)
	context.unsavedCache.balance[*tokenTypeId] = currentBalance
}

func (context *VmContext) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	currentBalance := context.GetBalance(context.address, tokenTypeId)

	newCurrentBalance := &big.Int{}
	newCurrentBalance.SetBytes(currentBalance.Bytes())
	newCurrentBalance.Sub(currentBalance, amount)

	if newCurrentBalance.Cmp(big.NewInt(0)) < 0 {
		return
	}
	context.unsavedCache.balance[*tokenTypeId] = newCurrentBalance
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
	context.unsavedCache = NewUnsavedCache(context.trie)
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
	context.unsavedCache.logList = append(context.unsavedCache.logList, log)
}

func (context *VmContext) GetLogListHash() *types.Hash {
	return context.unsavedCache.logList.Hash()
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
