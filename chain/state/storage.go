package chain_state

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func (sDB *StateDB) NewStorageDatabase(snapshotHash types.Hash, addr types.Address) (*StorageDatabase, error) {
	snapshotHeight, err := sDB.chain.GetSnapshotHeightByHash(snapshotHash)
	if err != nil {
		return nil, err
	}

	return NewStorageDatabase(sDB, snapshotHeight, addr), nil
}

type StorageDatabase struct {
	stateDb        *StateDB
	snapshotHash   types.Hash
	snapshotHeight uint64
	addr           types.Address
}

func NewStorageDatabase(stateDb *StateDB, snapshotHeight uint64, addr types.Address) *StorageDatabase {
	return &StorageDatabase{
		stateDb:        stateDb,
		snapshotHeight: snapshotHeight,
		addr:           addr,
	}
}

func (sd *StorageDatabase) GetValue(key []byte) ([]byte, error) {
	sd.stateDb.GetSnapshotValue(sd.snapshotHeight, sd.addr, key)
}

func (sd *StorageDatabase) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	ss, err := sd.stateDb.NewSnapshotStorageIteratorByHeight(sd.snapshotHeight, &sd.addr, prefix)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHeight is %s, addr is %s, prefix is %s",
			sd.snapshotHeight, sd.addr, prefix))
		return nil, cErr
	}

	if ss == nil {
		return nil, nil
	}
	return nil, nil
}

func (sd *StorageDatabase) Address() types.Address {
	return sd.addr
}
