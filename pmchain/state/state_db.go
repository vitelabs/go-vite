package chain_state

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/state/mvdb"
	"github.com/vitelabs/go-vite/pmchain/utils"
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

	ssm, err := newStateSnapshotManager(chain, mvDB)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		mvDB:  mvDB,
		chain: chain,
		ssm:   ssm,
	}, nil
}

func (sDB *StateDB) Destroy() error {
	if err := sDB.mvDB.Destroy(); err != nil {
		return err
	}

	sDB.mvDB = nil
	return nil
}

func (sDB *StateDB) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	accountId, err := sDB.chain.GetAccountId(addr)
	if err != nil {
		return nil, err
	}
	if accountId <= 0 {
		return nil, errors.New(fmt.Sprintf("account is not exsited, addr is %s", addr))
	}

	value, err := sDB.mvDB.GetValue(chain_utils.CreateBalanceKey(accountId, tokenTypeId))
	if err != nil {
		return nil, err
	}
	balance := big.NewInt(0).SetBytes(value)
	return balance, nil
}

func (sDB *StateDB) GetCode(addr *types.Address) ([]byte, error) {
	accountId, err := sDB.chain.GetAccountId(addr)
	if err != nil {
		return nil, err
	}
	if accountId <= 0 {
		return nil, errors.New(fmt.Sprintf("account is not exsited, addr is %s", addr))
	}

	return sDB.mvDB.GetValue(chain_utils.CreateCodeKey(accountId))
}

func (sDB *StateDB) GetContractMeta(addr *types.Address) (*ledger.ContractMeta, error) {
	accountId, err := sDB.chain.GetAccountId(addr)
	if err != nil {
		return nil, err
	}
	if accountId <= 0 {
		return nil, errors.New(fmt.Sprintf("account is not exsited, addr is %s", addr))
	}

	value, err := sDB.mvDB.GetValue(chain_utils.CreateContractMetaKey(accountId))
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
	accountId, err := sDB.chain.GetAccountId(addr)
	if err != nil {
		return false, err
	}
	if accountId <= 0 {
		return false, errors.New(fmt.Sprintf("account is not exsited, addr is %s", addr))
	}

	return sDB.mvDB.HasValue(chain_utils.CreateContractMetaKey(accountId))
}
func (sDB *StateDB) GetValue(addr *types.Address, key []byte) ([]byte, error) {
	accountId, err := sDB.chain.GetAccountId(addr)
	if err != nil {
		return nil, err
	}
	if accountId <= 0 {
		return nil, errors.New(fmt.Sprintf("account is not exsited, addr is %s", addr))
	}

	return sDB.mvDB.GetValue(chain_utils.CreateStorageKeyPrefix(accountId, []byte(key)))
}

func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	accountId, err := sDB.chain.GetAccountId(addr)
	if err != nil {
		return nil, err
	}
	if accountId <= 0 {
		return nil, errors.New(fmt.Sprintf("account is not exsited, addr is %s", addr))
	}

	return sDB.mvDB.NewIterator(chain_utils.CreateStorageKeyPrefix(accountId, prefix)), nil
}

func (sDB *StateDB) NewStateSnapshot(addr *types.Address, snapshotBlockHeight uint64) (interfaces.StateSnapshot, error) {
	return sDB.ssm.NewStateSnapshot(addr, snapshotBlockHeight)
}
