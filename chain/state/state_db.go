package chain_state

import (
	"bytes"
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/pending"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
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

func (sDB *StateDB) NewIterator(slice *util.Range) interfaces.StorageIterator {
	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
		sDB.pending.NewIterator(slice),
		sDB.db.NewIterator(slice, nil),
	}, sDB.pending.IsDelete)
}

func (sDB *StateDB) GetValue(key []byte) ([]byte, error) {
	if value, ok := sDB.pending.Get(key); ok {
		return value, nil
	}

	value, err := sDB.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return value, nil
}

func (sDB *StateDB) hasValue(key []byte) (bool, error) {
	if ok, deleted := sDB.pending.Has(key); ok {
		return ok, nil
	} else if deleted {
		return false, nil

	}

	return sDB.db.Has(key, nil)
}

func (sDB *StateDB) QueryLatestLocation() (*chain_file_manager.Location, error) {
	value, err := sDB.GetValue(chain_utils.CreateStateDbLocationKey())
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
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

func (sDB *StateDB) GetStorageValue(addr *types.Address, key []byte) ([]byte, error) {
	value, err := sDB.GetValue(chain_utils.CreateStorageValueKey(addr, key))
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (sDB *StateDB) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	value, err := sDB.GetValue(chain_utils.CreateBalanceKey(addr, tokenTypeId))
	if err != nil {
		return nil, err
	}
	balance := big.NewInt(0).SetBytes(value)
	return balance, nil
}

func (sDB *StateDB) GetBalanceMap(addr *types.Address) (map[types.TokenTypeId]*big.Int, error) {
	balanceMap := make(map[types.TokenTypeId]*big.Int)
	iter := sDB.NewIterator(util.BytesPrefix(chain_utils.CreateBalanceKeyPrefix(addr)))
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		tokenTypeIdBytes := key[1+types.AddressSize : 1+types.AddressSize+types.TokenTypeIdSize]
		tokenTypeId, err := types.BytesToTokenTypeId(tokenTypeIdBytes)
		if err != nil {
			return nil, err
		}
		balanceMap[tokenTypeId] = big.NewInt(0).SetBytes(iter.Value())
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return balanceMap, nil
}

func (sDB *StateDB) GetCode(addr *types.Address) ([]byte, error) {
	code, err := sDB.GetValue(chain_utils.CreateCodeKey(addr))
	if err != nil {
		return nil, err
	}

	return code, nil
}

//
func (sDB *StateDB) GetContractMeta(addr *types.Address) (*ledger.ContractMeta, error) {
	value, err := sDB.GetValue(chain_utils.CreateContractMetaKey(addr))
	if err != nil {
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
	return sDB.hasValue(chain_utils.CreateContractMetaKey(addr))
}

func (sDB *StateDB) GetContractList(gid *types.Gid) ([]types.Address, error) {
	iter := sDB.NewIterator(util.BytesPrefix(chain_utils.CreateGidContractPrefixKey(gid)))
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
	value, err := sDB.GetValue(chain_utils.CreateVmLogListKey(logHash))
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return ledger.VmLogListDeserialize(value)
}

func (sDB *StateDB) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	value, err := sDB.GetValue(chain_utils.CreateCallDepthKey(sendBlockHash))
	if err != nil {
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

// TODO
func (sDB *StateDB) GetSnapshotValue(snapshotBlockHeight uint64, addr types.Address, key []byte) ([]byte, error) {
	var iter iterator.Iterator

	startHistoryStorageKey := chain_utils.CreateHistoryStorageValueKey(&addr, key, 0)
	endHistoryStorageKey := chain_utils.CreateHistoryStorageValueKey(&addr, key, snapshotBlockHeight+1)

	iter = sDB.db.NewIterator(&util.Range{Start: startHistoryStorageKey, Limit: endHistoryStorageKey}, nil)
	defer iter.Release()

	if iter.Last() {
		return iter.Value(), nil
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return nil, nil
}
