package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
)

const (
	GenesisKey = byte(0)
)

func (c *chain) WriteGenesisCheckSum(hash types.Hash) error {
	if err := c.metaDB.Put([]byte{GenesisKey}, hash.Bytes(), nil); err != nil {
		return err
	}
	return nil
}

func (c *chain) QueryGenesisCheckSum() (*types.Hash, error) {
	value, err := c.metaDB.Get([]byte{GenesisKey}, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}

	checkSum, err := types.BytesToHash(value)
	if err != nil {
		return nil, err
	}
	return &checkSum, nil
}
