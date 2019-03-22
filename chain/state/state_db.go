package chain_state

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/state/mvdb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type StateDB struct {
	mvDB  *mvdb.MultiVersionDB
	chain Chain

	ssm *stateSnapshotManager
}

func NewStateDB(chain Chain, chainDir string) (*StateDB, error) {
	mvDB, err := mvdb.NewMultiVersionDB(chainDir)
	if err != nil {
		return nil, err
	}

	return &StateDB{
		mvDB:  mvDB,
		chain: chain,
	}, nil
}

func (sDB *StateDB) Init() error {
	ssm, err := newStateSnapshotManager(sDB.chain, sDB.mvDB)
	if err != nil {
		return err
	}
	sDB.ssm = ssm

	return nil
}

func (sDB *StateDB) QueryLatestLocation() (*chain_block.Location, error) {
	return sDB.mvDB.QueryLatestLocation()
}

func (sDB *StateDB) Destroy() error {
	if err := sDB.mvDB.Destroy(); err != nil {
		return err
	}

	sDB.mvDB = nil
	return nil
}

func (sDB *StateDB) GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	key := chain_utils.CreateVmLogListKey(logHash)
	value, err := sDB.mvDB.GetValue(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return ledger.VmLogListDeserialize(value)
}

func (sDB *StateDB) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	value, err := sDB.mvDB.GetValue(chain_utils.CreateBalanceKey(addr, tokenTypeId))
	if err != nil {
		return nil, err
	}
	balance := big.NewInt(0).SetBytes(value)
	return balance, nil
}

func (sDB *StateDB) GetCode(addr *types.Address) ([]byte, error) {

	return sDB.mvDB.GetValue(chain_utils.CreateCodeKey(addr))
}

func (sDB *StateDB) GetContractMeta(addr *types.Address) (*ledger.ContractMeta, error) {
	value, err := sDB.mvDB.GetValue(chain_utils.CreateContractMetaKey(addr))
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
	return sDB.mvDB.HasValue(chain_utils.CreateContractMetaKey(addr))
}
func (sDB *StateDB) GetValue(addr *types.Address, key []byte) ([]byte, error) {
	return sDB.mvDB.GetValue(chain_utils.CreateStorageKeyPrefix(addr, []byte(key)))
}

func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	return sDB.mvDB.NewIterator(chain_utils.CreateStorageKeyPrefix(addr, prefix)), nil
}

func (sDB *StateDB) NewStateSnapshot(addr *types.Address, snapshotBlockHeight uint64) (interfaces.StateSnapshot, error) {
	return sDB.ssm.NewStateSnapshot(addr, snapshotBlockHeight)
}
