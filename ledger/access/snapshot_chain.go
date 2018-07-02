package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotChainAccess struct {
	store *vitedb.SnapshotChain
}

var _snapshotChainAccess *SnapshotChainAccess


func GetSnapshotChainAccess () *SnapshotChainAccess {
	if _snapshotChainAccess == nil {
		_snapshotChainAccess = &SnapshotChainAccess {
			store: vitedb.GetSnapshotChain(),
		}
	}
	return _snapshotChainAccess
}

func (sca *SnapshotChainAccess) GetBlockByHash (blockHash []byte) (*ledger.SnapshotBlock, error) {
	block, err:= sca.store.GetBlockByHash(blockHash)

	if err != nil {
		return nil, err
	}

	return block, nil
}


func (sca *SnapshotChainAccess) GetBlockList (index int, num int, count int) ([]*ledger.SnapshotBlock, error) {
	blockList, err:= sca.store.GetBlockList(index, num, count)

	if err != nil {
		return nil, err
	}

	return blockList, nil
}