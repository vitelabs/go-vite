package chain_state

import (
	"bytes"
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"path"
	"sync"
)

type StateDB struct {
	chain Chain
	store *chain_db.Store

	log log15.Logger

	storageRedo *StorageRedo

	historyKv   map[types.Hash][][2][]byte
	historyKvMu sync.Mutex
}

func NewStateDB(chain Chain, chainDir string) (*StateDB, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("stateDb")))

	var err error
	store, err := chain_db.NewStore(path.Join(chainDir, "state"), id)

	if err != nil {
		return nil, err
	}

	storageRedo, err := NewStorageRedo(chain, chainDir)
	if err != nil {
		return nil, err
	}

	stateDb := &StateDB{
		chain: chain,
		log:   log15.New("module", "stateDB"),
		store: store,

		storageRedo: storageRedo,
		historyKv:   make(map[types.Hash][][2][]byte),
	}

	return stateDb, nil
}

func (sDB *StateDB) Close() error {
	if err := sDB.store.Close(); err != nil {
		return err
	}

	sDB.store = nil

	if err := sDB.storageRedo.Close(); err != nil {
		return err
	}
	sDB.storageRedo = nil
	return nil
}

func (sDB *StateDB) GetStorageValue(addr *types.Address, key []byte) ([]byte, error) {
	value, err := sDB.store.Get(chain_utils.CreateStorageValueKey(addr, key))
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (sDB *StateDB) GetBalance(addr types.Address, tokenTypeId types.TokenTypeId) (*big.Int, error) {
	value, err := sDB.store.Get(chain_utils.CreateBalanceKey(addr, tokenTypeId))
	if err != nil {
		return nil, err
	}
	balance := big.NewInt(0).SetBytes(value)
	return balance, nil
}

func (sDB *StateDB) GetBalanceMap(addr types.Address) (map[types.TokenTypeId]*big.Int, error) {
	balanceMap := make(map[types.TokenTypeId]*big.Int)
	iter := sDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateBalanceKeyPrefix(addr)))
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

func (sDB *StateDB) GetCode(addr types.Address) ([]byte, error) {
	code, err := sDB.store.Get(chain_utils.CreateCodeKey(addr))
	if err != nil {
		return nil, err
	}

	return code, nil
}

//
func (sDB *StateDB) GetContractMeta(addr types.Address) (*ledger.ContractMeta, error) {
	value, err := sDB.store.Get(chain_utils.CreateContractMetaKey(addr))
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

func (sDB *StateDB) HasContractMeta(addr types.Address) (bool, error) {
	return sDB.store.Has(chain_utils.CreateContractMetaKey(addr))
}

func (sDB *StateDB) GetContractList(gid *types.Gid) ([]types.Address, error) {
	iter := sDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateGidContractPrefixKey(gid)))
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
	value, err := sDB.store.Get(chain_utils.CreateVmLogListKey(logHash))
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return ledger.VmLogListDeserialize(value)
}

func (sDB *StateDB) GetCallDepth(sendBlockHash *types.Hash) (uint16, error) {
	value, err := sDB.store.Get(chain_utils.CreateCallDepthKey(sendBlockHash))
	if err != nil {
		return 0, err
	}

	if len(value) <= 0 {
		return 0, nil
	}

	return binary.BigEndian.Uint16(value), nil
}

func (sDB *StateDB) GetSnapshotBalanceList(snapshotBlockHash types.Hash, addrList []types.Address, tokenId types.TokenTypeId) (map[types.Address]*big.Int, error) {
	balanceMap := make(map[types.Address]*big.Int, len(addrList))

	prefix := chain_utils.BalanceHistoryKeyPrefix

	iter := sDB.store.NewIterator(util.BytesPrefix([]byte{prefix}))
	defer iter.Release()

	snapshotHeight, err := sDB.chain.GetSnapshotHeightByHash(snapshotBlockHash)
	if err != nil {
		return nil, err
	}

	seekKey := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize+8)
	seekKey[0] = prefix

	copy(seekKey[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize], tokenId.Bytes())
	binary.BigEndian.PutUint64(seekKey[1+types.AddressSize+types.TokenTypeIdSize:], snapshotHeight+1)

	for _, addr := range addrList {
		copy(seekKey[1:1+types.AddressSize], addr.Bytes())

		iter.Seek(seekKey)

		ok := iter.Prev()
		if !ok {
			continue
		}

		key := iter.Key()
		if bytes.HasPrefix(key, seekKey[:len(seekKey)-8]) {
			balanceMap[addr] = big.NewInt(0).SetBytes(iter.Value())
			//FOR DEBUG
			//fmt.Println("query", addr, balanceMap[addr], binary.BigEndian.Uint64(key[len(key)-8:]))
		}

	}
	return balanceMap, nil
}

func (sDB *StateDB) GetSnapshotValue(snapshotBlockHeight uint64, addr types.Address, key []byte) ([]byte, error) {

	startHistoryStorageKey := chain_utils.CreateHistoryStorageValueKey(&addr, key, 0)
	endHistoryStorageKey := chain_utils.CreateHistoryStorageValueKey(&addr, key, snapshotBlockHeight+1)

	iter := sDB.store.NewIterator(&util.Range{Start: startHistoryStorageKey, Limit: endHistoryStorageKey})
	defer iter.Release()

	if iter.Last() {
		return iter.Value(), nil
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return nil, nil
}

func (sDB *StateDB) Store() *chain_db.Store {
	return sDB.store
}
func (sDB *StateDB) StorageRedo() *StorageRedo {
	return sDB.storageRedo
}
