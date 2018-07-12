package service

import (
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotChain struct {}

func (*SnapshotChain) GetBlockByHash (blockHash *[]byte, block * ledger.SnapshotBlock) error {
	return nil
}

func (*SnapshotChain) GetBlockList (blockQuery *SnapshotBlockQuery, blockList []*ledger.SnapshotBlock) error {
	return nil
}