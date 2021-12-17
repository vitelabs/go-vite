package chain_state

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (sDB *StateDB) NewStorageDatabase(snapshotHash types.Hash, addr types.Address) (StorageDatabaseInterface, error) {
	snapshotHeight, err := sDB.chain.GetSnapshotHeightByHash(snapshotHash)
	if err != nil {
		return nil, err
	}
	if snapshotHeight <= 0 {
		return nil, fmt.Errorf("snapshot hash %s is not existed", snapshotHash)
	}

	return NewStorageDatabase(sDB, ledger.HashHeight{
		Height: snapshotHeight,
		Hash:   snapshotHash,
	}, addr), nil
}

type StorageDatabase struct {
	stateDb        *StateDB
	snapshotHash   types.Hash
	snapshotHeight uint64
	addr           types.Address
}

func NewStorageDatabase(stateDb *StateDB, hashHeight ledger.HashHeight, addr types.Address) StorageDatabaseInterface {
	return &StorageDatabase{
		stateDb:        stateDb,
		snapshotHeight: hashHeight.Height,
		snapshotHash:   hashHeight.Hash,
		addr:           addr,
	}
}

func (sd *StorageDatabase) GetValue(key []byte) ([]byte, error) {
	return sd.stateDb.GetSnapshotValue(sd.snapshotHeight, sd.addr, key)
}

func (sd *StorageDatabase) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	// if use cache
	if sd.stateDb.consensusCacheLevel == ConsensusReadCache &&
		sd.addr == types.AddressGovernance {
		if iter := sd.stateDb.roundCache.StorageIterator(sd.snapshotHash); iter != nil {
			return iter, nil
		}

	}
	ss, err := sd.stateDb.NewSnapshotStorageIteratorByHeight(sd.snapshotHeight, sd.addr, prefix)
	if err != nil {
		cErr := fmt.Errorf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHeight is %d, addr is %s, prefix is %s",
			sd.snapshotHeight, sd.addr, prefix)
		return nil, cErr
	}

	return ss, nil
}

func (sd *StorageDatabase) Address() *types.Address {
	return &sd.addr
}
