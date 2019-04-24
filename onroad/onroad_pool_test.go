package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type TestChainDB struct {
}

func (db *TestChainDB) LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error) {
	return nil, nil
}

func (db *TestChainDB) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *TestChainDB) IsContractAccount(address types.Address) (bool, error) {
	return true, nil
}

func (db *TestChainDB) GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}
