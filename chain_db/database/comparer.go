package database

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"math/big"
)

type DbComparer struct {
	name string
}

func NewDbComparer(name string) *DbComparer {
	return &DbComparer{
		name: name,
	}

}

func (c *DbComparer) Name() string {
	return c.name
}

func (*DbComparer) Separator(dst, a, b []byte) []byte {
	return comparer.DefaultComparer.Separator(dst, a, b)
}

func (*DbComparer) Successor(dst, b []byte) []byte {
	return comparer.DefaultComparer.Successor(dst, b)
}

func (*DbComparer) Compare(a, b []byte) (result int) {
	aKey, _ := DecodeKey(a)
	bKey, _ := DecodeKey(b)
	if aKey.Prefix > bKey.Prefix {
		return 1
	} else if aKey.Prefix < bKey.Prefix {
		return -1
	}

	for index, aKeyPartion := range aKey.KeyPartionList {
		bKeyPartion := bKey.KeyPartionList[index]

		if bKeyPartion == nil {
			return 1
		}

		aDataType := aKeyPartion[0]
		bDataType := bKeyPartion[0]
		if aDataType == KEY_MAX {
			return 1
		} else if bDataType == KEY_MAX {
			return -1
		}

		if aDataType == KEY_BYTE_SLICE {
			if bDataType == KEY_BIG_INT {
				return 1
			}

			result := bytes.Compare(aKeyPartion[1:], bKeyPartion[1:])

			if result != 0 {
				return result
			}
		} else {
			if bDataType == KEY_BYTE_SLICE {
				return -1
			}

			aBigInt := &big.Int{}
			aBigInt.SetBytes(aKeyPartion[1:])

			bBigInt := &big.Int{}
			bBigInt.SetBytes(bKeyPartion[1:])

			result := aBigInt.Cmp(bBigInt)

			if result != 0 {
				return result
			}
		}
	}

	if len(aKey.KeyPartionList) < len(bKey.KeyPartionList) {
		return -1
	} else {
		return 0
	}
}
