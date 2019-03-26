package chain_state

import (
	"bytes"
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/pending"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"path"
)

type StateDB struct {
	chain Chain
	db    *leveldb.DB

	pending *chain_pending.MemDB

	undoLogger *undoLogger

	log log15.Logger
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
		log:        log15.New("module", "stateDB"),
		db:         db,
		pending:    chain_pending.NewMemDB(),
		undoLogger: undoLogger,
	}, nil
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

func (sDB *StateDB) GetContractList(gid *types.Gid) ([]types.Address, error) {
	iter := sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateGidContractPrefixKey(gid)), nil)
	defer iter.Release()

	var contractList []types.Address
	for iter.Next() {
		key := iter.Key()

		addr, err := types.BytesToAddress(key[len(key)-types.AddressSize:])
		if err != nil {
			return nil, err
		}
		contractList = append(contractList, addr)
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return contractList, nil
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

func (sDB *StateDB) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	value, err := sDB.db.Get(chain_utils.CreateCallDepthKey(sendBlockHash), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}

	if len(value) <= 0 {
		return 0, nil
	}

	return binary.BigEndian.Uint16(value), nil
}

// TODO

func (sDB *StateDB) GetSnapshotBalanceList(snapshotBlockHash types.Hash, addrList []types.Address, tokenId types.TokenTypeId) (map[types.Address]*big.Int, error) {
	var iter iterator.Iterator
	balanceMap := make(map[types.Address]*big.Int, len(addrList))

	prefix := chain_utils.BalanceHistoryKeyPrefix
	if snapshotBlockHash == sDB.chain.GetLatestSnapshotBlock().Hash {
		prefix = chain_utils.BalanceKeyPrefix
	}

	iter = sDB.db.NewIterator(util.BytesPrefix([]byte{prefix}), nil)
	defer iter.Release()

	snapshotHeight, err := sDB.chain.GetSnapshotHeightByHash(snapshotBlockHash)
	if err != nil {
		return nil, err
	}

	seekKey := make([]byte, types.AddressSize+42)
	seekKey[0] = prefix
	binary.BigEndian.PutUint64(seekKey[len(seekKey)-8:], snapshotHeight)

	for _, addr := range addrList {
		copy(seekKey[1:1+types.AddressSize], addr.Bytes())
		copy(seekKey[2+types.AddressSize:2+types.AddressSize+types.TokenTypeIdSize], tokenId.Bytes())

		ok := iter.Seek(seekKey)
		if !ok {
			continue
		}

		key := iter.Key()
		if bytes.HasPrefix(key, seekKey[:len(seekKey)-8]) {
			balanceMap[addr] = big.NewInt(0).SetBytes(iter.Value())
		}

	}
	return balanceMap, nil
}
