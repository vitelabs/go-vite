package chain_state_bak

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"path"
)

type DemoDB struct {
	db *leveldb.DB
}

func NewDemoDB(chainDir string) (*DemoDB, error) {
	dbDir := path.Join(chainDir, "demo")

	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}

	return &DemoDB{
		db,
	}, nil
}

func (demoDB *DemoDB) Write(key, value []byte, snapshotHeight uint64) error {
	innerKey := ToInnerKey(key, snapshotHeight)
	return demoDB.db.Put(innerKey, value, nil)
}

func ToInnerKey(key []byte, snapshotHeight uint64) []byte {
	newKey := make([]byte, 41)
	copy(newKey, key)
	newKey[32] = byte(len(key))
	binary.BigEndian.PutUint64(newKey[33:], snapshotHeight)
	return newKey
}
func (demoDB *DemoDB) Destory() error {
	return demoDB.db.Close()
}
