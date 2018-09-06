package helper

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
)

const (
	KEY_MAX = byte(iota)
	KEY_BYTE_SLICE
	KEY_BIG_INT
)

func DecodeKey(buf []byte) (*vitepb.DbKey, error) {
	key := &vitepb.DbKey{}
	if err := proto.Unmarshal(buf, key); err != nil {
		return nil, err
	}

	return key, nil
}

func EncodeKey(prefix uint32, partionList ...interface{}) ([]byte, error) {
	dbKey := &vitepb.DbKey{
		Prefix:         prefix,
		KeyPartionList: make([][]byte, 0, len(partionList)),
	}

	for _, partion := range partionList {
		var buffer []byte
		var src []byte
		switch partion.(type) {
		case *big.Int:
			src = partion.(*big.Int).Bytes()
			if len(src) <= 0 {
				src = []byte{0}
			}
			buffer = append(buffer, KEY_BIG_INT)
		case []byte:
			src = partion.([]byte)
			buffer = append(buffer, KEY_BYTE_SLICE)
		case string:
			if partion.(string) == "KEY_MAX" {
				buffer = append(buffer, KEY_MAX)
			}
		}

		buffer = append(buffer, src...)

		dbKey.KeyPartionList = append(dbKey.KeyPartionList, buffer)
	}

	return proto.Marshal(dbKey)
}
