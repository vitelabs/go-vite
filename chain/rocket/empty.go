package rocket

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"time"
)

func (c *chain) GetAccount(address *types.Address) (*ledger.Account, error) {
	return nil, nil
}
func (c *chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (c *chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return nil
}
func (c *chain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}
func (c *chain) GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}
func (c *chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}
func (c *chain) GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (c *chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}
func (c *chain) GetStateTrie(hash *types.Hash) *trie.Trie {
	return nil
}

func (c *chain) NewStateTrie() *trie.Trie {
	return nil
}

func (c *chain) GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}
func (c *chain) GetContractGid(addr *types.Address) (*types.Gid, error) {
	return nil, nil
}

func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {
	return nil
}

func (c *chain) GetNeedSnapshotContent() ledger.SnapshotContent {
	return nil
}

func (c *chain) GenStateTrie(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error) {
	return nil, nil
}
