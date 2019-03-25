package chain_state

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/pending"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"path"
)

type StateDB struct {
	chain Chain
	db    *leveldb.DB

	pending *chain_pending.MemDB

	undoLogger *undoLogger
}

func NewStateDB(chain Chain, chainDir string) (*StateDB, error) {
	dbDir := path.Join(chainDir, "state")

	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}

	undoLogger, err := newUndoLogger(chainDir)
	if err != nil {
		return nil, err
	}

	return &StateDB{
		chain:      chain,
		db:         db,
		pending:    chain_pending.NewMemDB(),
		undoLogger: undoLogger,
	}, nil
}

func (sDB *StateDB) Init() error {
	//	ssm, err := newStateSnapshotManager(sDB.chain, sDB.mvDB)
	//	if err != nil {
	//		return err
	//	}
	//	sDB.ssm = ssm
	//
	return nil
}

func (sDB *StateDB) QueryLatestLocation() (*chain_block.Location, error) {
	value, err := sDB.db.Get(chain_utils.CreateStateDbLocationKey(), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return chain_utils.DeserializeLocation(value), nil
}

func (sDB *StateDB) Destroy() error {
	if err := sDB.db.Close(); err != nil {
		return err
	}

	sDB.db = nil
	return nil
}

func (sDB *StateDB) GetValue(addr *types.Address, key []byte) ([]byte, error) {
	value, err := sDB.db.Get(chain_utils.CreateStorageValueKey(addr, key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return value, nil
}

// TODO
func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	return sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKey(addr, prefix)), nil), nil
}

func (sDB *StateDB) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	value, err := sDB.db.Get(chain_utils.CreateBalanceKey(addr, tokenTypeId), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	balance := big.NewInt(0).SetBytes(value)
	return balance, nil
}

func (sDB *StateDB) GetCode(addr *types.Address) ([]byte, error) {
	code, err := sDB.db.Get(chain_utils.CreateCodeKey(addr), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	return code, nil
}

//
func (sDB *StateDB) GetContractMeta(addr *types.Address) (*ledger.ContractMeta, error) {
	value, err := sDB.db.Get(chain_utils.CreateContractMetaKey(addr), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}

	meta := &ledger.ContractMeta{}
	if err := meta.Deserialize(value); err != nil {
		return nil, err
	}
	return meta, nil
}

func (sDB *StateDB) HasContractMeta(addr *types.Address) (bool, error) {
	return sDB.db.Has(chain_utils.CreateContractMetaKey(addr), nil)
}

func (sDB *StateDB) GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	value, err := sDB.db.Get(chain_utils.CreateVmLogListKey(logHash), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return ledger.VmLogListDeserialize(value)
}

func (sDB *StateDB) NewSnapshotStorageIterator(snapshotHash *types.Hash, addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	//return sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKey(addr, prefix)), nil), nil
	return nil, nil
}

func (sDB *StateDB) GetSnapshotValue(snapshotHash *types.Hash, addr *types.Address, key []byte) ([]byte, error) {
	//return sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKey(addr, prefix)), nil), nil
	return nil, nil
}

// TODO
func (sDB *StateDB) GetSnapshotBalance(addr *types.Address, tokenTypeId *types.TokenTypeId, snapshotHash *types.Hash) (*big.Int, error) {
	return nil, nil
}

//func (sDB *StateDB) NewStateSnapshot(addr *types.Address, snapshotBlockHeight uint64) (interfaces.StateSnapshot, error) {
//	return sDB.ssm.NewStateSnapshot(addr, snapshotBlockHeight)
//}
