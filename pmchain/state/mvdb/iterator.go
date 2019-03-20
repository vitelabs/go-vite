package mvdb

import (
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/pmchain/utils"
)

func (mvDB *MultiVersionDB) NewIterator(prefix []byte) interfaces.StorageIterator {
	keyIdIterator := mvDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateKeyIdKey(prefix)), nil)
	return NewIterator(mvDB, keyIdIterator)
}

func (mvDB *MultiVersionDB) NewSnapshotIterator(prefix []byte, isKeyIdValid func(uint64) bool, getValueIdFromCache func(uint64) (uint64, bool)) interfaces.StorageIterator {
	keyIdIterator := mvDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateKeyIdKey(prefix)), nil)
	return NewSnapshotIterator(mvDB, keyIdIterator, isKeyIdValid, getValueIdFromCache)
}

type Iterator struct {
	mvDB          *MultiVersionDB
	keyIdIterator interfaces.StorageIterator

	currentKeyId uint64
	currentKey   []byte

	currentValueId uint64
	currentValue   []byte

	err error

	isKeyIdValid        func(uint64) bool
	getValueIdFromCache func(uint64) (uint64, bool)
}

func NewIterator(mvDB *MultiVersionDB, keyIdIterator interfaces.StorageIterator) *Iterator {
	return &Iterator{
		mvDB:          mvDB,
		keyIdIterator: keyIdIterator,
	}
}

func NewSnapshotIterator(mvDB *MultiVersionDB, keyIdIterator interfaces.StorageIterator,
	isKeyIdValid func(uint64) bool, getValueIdFromCache func(uint64) (uint64, bool)) *Iterator {

	return &Iterator{
		mvDB:          mvDB,
		keyIdIterator: keyIdIterator,

		isKeyIdValid:        isKeyIdValid,
		getValueIdFromCache: getValueIdFromCache,
	}
}

func (iterator *Iterator) Next() bool {
	return iterator.step(true)
}

func (iterator *Iterator) Key() []byte {
	return iterator.currentKey
}
func (iterator *Iterator) Value() []byte {
	if iterator.err != nil {
		return nil
	}

	if iterator.currentValue != nil {
		return iterator.currentValue
	}
	if iterator.currentValueId <= 0 {
		return nil
	}

	iterator.currentValue, _ = iterator.mvDB.GetValueByValueId(iterator.currentValueId)
	return iterator.currentValue
}
func (iterator *Iterator) Error() error {
	return iterator.err
}

func (iterator *Iterator) Release() {
	iterator.currentKeyId = 0
	iterator.currentKey = nil

	iterator.currentValueId = 0
	iterator.currentValue = nil

	iterator.err = nil

	iterator.mvDB = nil
	iterator.keyIdIterator.Release()
}

func (iterator *Iterator) setError(err error) {
	iterator.currentKeyId = 0
	iterator.currentKey = nil

	iterator.currentValueId = 0
	iterator.currentValue = nil

	iterator.err = err
}

func (iterator *Iterator) step(next bool) bool {
	if iterator.err != nil {
		return false
	}

	iterator.currentValueId = 0
	iterator.currentValue = nil

	for {
		iterator.currentKeyId = 0
		iterator.currentKey = nil

		if ok := iterator.keyIdIterator.Next(); !ok {
			iterator.setError(iterator.keyIdIterator.Error())
			return false
		}
		var err error
		iterator.currentKey = iterator.Key()[1:]
		iterator.currentKeyId, err = iterator.mvDB.GetKeyId(iterator.currentKey)
		if iterator.isKeyIdValid != nil {
			if !iterator.isKeyIdValid(iterator.currentKeyId) {
				continue
			}
		}
		if err != nil {
			iterator.setError(err)
			return false
		}

		var valueId uint64
		gotValueId := false

		if iterator.getValueIdFromCache != nil {
			valueId, gotValueId = iterator.getValueIdFromCache(iterator.currentKeyId)
		}
		if !gotValueId {
			var err error
			valueId, err = iterator.mvDB.GetValueId(iterator.currentKeyId)
			if err != nil {
				iterator.setError(err)
				return false
			}
		}

		if valueId > 0 {
			iterator.currentValueId = valueId
			break
		}
	}

	return true
}
