//

package rpc

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

// LedgerApi ...
type LedgerApi interface {
	GetRawBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetBlockByHash(blockHash types.Hash) (*api.AccountBlock, error)
	GetCompleteBlockByHash(blockHash types.Hash) (*api.AccountBlock, error)
	GetBlocksByHash(addr types.Address, originBlockHash *types.Hash, count uint64) ([]*api.AccountBlock, error)
	GetVmLogListByHash(logHash types.Hash) (ledger.VmLogList, error)
	GetBlocksByHeight(addr types.Address, height interface{}, count uint64) ([]*api.AccountBlock, error)
	GetBlockByHeight(addr types.Address, height interface{}) (*api.AccountBlock, error)
	GetBlocksByAccAddr(addr types.Address, index int, count int) ([]*api.AccountBlock, error)
	GetAccountByAccAddr(addr types.Address) (*api.RpcAccountInfo, error)
	GetSnapshotBlockByHash(hash types.Hash) (*api.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height interface{}) (*api.SnapshotBlock, error)
	GetSnapshotBlocks(height interface{}, count int) ([]*api.SnapshotBlock, error)
	GetChunks(startHeight interface{}, endHeight interface{}) ([]*api.SnapshotChunk, error)
	GetSnapshotChainHeight() string
	GetLatestSnapshotChainHash() *types.Hash
	GetLatestBlock(addr types.Address) (*api.AccountBlock, error)
	GetVmLogList(blockHash types.Hash) (ledger.VmLogList, error)
	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock
	GetConfirmedBalances(snapshotHash types.Hash, addrList []types.Address, tokenIds []types.TokenTypeId) (api.GetBalancesRes, error)
	GetHourSBPStats(startIdx uint64, endIdx uint64) ([]map[string]interface{}, error)
}

type ledgerApi struct {
	cc *rpc.Client
}

func NewLedgerApi(cc *rpc.Client) LedgerApi {
	return &ledgerApi{cc: cc}
}

func (li ledgerApi) GetRawBlockByHash(blockHash types.Hash) (block *ledger.AccountBlock, err error) {
	block = &ledger.AccountBlock{}
	err = li.cc.Call(block, "ledger_getRawBlockByHash", blockHash)
	return
}

func (li ledgerApi) GetBlockByHash(blockHash types.Hash) (block *api.AccountBlock, err error) {
	block = &api.AccountBlock{}
	err = li.cc.Call(block, "ledger_getBlockByHash", blockHash)
	return
}

func (li ledgerApi) GetCompleteBlockByHash(blockHash types.Hash) (block *api.AccountBlock, err error) {
	block = &api.AccountBlock{}
	err = li.cc.Call(block, "ledger_getCompleteBlockByHash", blockHash)
	return
}

func (li ledgerApi) GetBlocksByHash(addr types.Address, originBlockHash *types.Hash, count uint64) (blocks []*api.AccountBlock, err error) {
	err = li.cc.Call(&blocks, "ledger_getBlocksByHash", addr, originBlockHash, count)
	return
}

func (li ledgerApi) GetVmLogListByHash(logHash types.Hash) (logs ledger.VmLogList, err error) {
	err = li.cc.Call(&logs, "ledger_getVmLogListByHash", logHash)
	return
}

func (li ledgerApi) GetBlocksByHeight(addr types.Address, height interface{}, count uint64) (blocks []*api.AccountBlock, err error) {
	err = li.cc.Call(&blocks, "ledger_getBlocksByHeight", height, count)
	return
}

func (li ledgerApi) GetBlockByHeight(addr types.Address, height interface{}) (block *api.AccountBlock, err error) {
	block = &api.AccountBlock{}
	err = li.cc.Call(block, "ledger_getBlockByHeight", addr, height)
	return
}

func (li ledgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int) (blocks []*api.AccountBlock, err error) {
	err = li.cc.Call(&blocks, "ledger_getBlocksByAccAddr", addr, index, count)
	return
}

func (li ledgerApi) GetAccountByAccAddr(addr types.Address) (block *api.RpcAccountInfo, err error) {
	block = &api.RpcAccountInfo{}
	err = li.cc.Call(block, "ledger_getAccountByAccAddr", addr)
	return
}

func (li ledgerApi) GetSnapshotBlockByHash(hash types.Hash) (block *api.SnapshotBlock, err error) {
	block = &api.SnapshotBlock{}
	err = li.cc.Call(block, "ledger_getSnapshotBlockByHash", hash)
	return
}

func (li ledgerApi) GetSnapshotBlockByHeight(height interface{}) (block *api.SnapshotBlock, err error) {
	block = &api.SnapshotBlock{}
	err = li.cc.Call(block, "ledger_getSnapshotBlockByHeight", height)
	return
}

func (li ledgerApi) GetSnapshotBlocks(height interface{}, count int) (blocks []*api.SnapshotBlock, err error) {
	err = li.cc.Call(&blocks, "ledger_getSnapshotBlocks", height, count)
	return
}

func (li ledgerApi) GetChunks(startHeight interface{}, endHeight interface{}) (blocks []*api.SnapshotChunk, err error) {
	err = li.cc.Call(&blocks, "ledger_getChunks", startHeight, endHeight)
	return
}

func (li ledgerApi) GetSnapshotChainHeight() (result string) {
	li.cc.Call(&result, "ledger_getSnapshotChainHeight")
	return
}

func (li ledgerApi) GetLatestSnapshotChainHash() (hash *types.Hash) {
	hash = &types.Hash{}
	li.cc.Call(hash, "ledger_getLatestSnapshotChainHash")
	return
}

func (li ledgerApi) GetLatestBlock(addr types.Address) (block *api.AccountBlock, err error) {
	block = &api.AccountBlock{}
	err = li.cc.Call(&block, "ledger_getLatestBlock", addr)
	return
}

func (li ledgerApi) GetVmLogList(blockHash types.Hash) (logs ledger.VmLogList, err error) {
	err = li.cc.Call(&logs, "ledger_getVmLogList", blockHash)
	return
}

func (li ledgerApi) GetUnconfirmedBlocks(addr types.Address) (blocks []*ledger.AccountBlock) {
	li.cc.Call(&blocks, "ledger_getUnconfirmedBlocks", addr)
	return
}

func (li ledgerApi) GetConfirmedBalances(snapshotHash types.Hash, addrList []types.Address, tokenIds []types.TokenTypeId) (result api.GetBalancesRes, err error) {
	err = li.cc.Call(&result, "ledger_getConfirmedBalances", snapshotHash, addrList, tokenIds)
	return
}

func (li ledgerApi) GetHourSBPStats(startIdx uint64, endIdx uint64) (result []map[string]interface{}, err error) {
	err = li.cc.Call(&result, "sbpstats_getHourSBPStats", startIdx, endIdx)
	return
}
