package database

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
)

const (
	KEY_BYTE_SLICE = byte(iota)
	KEY_BIG_INT

	KEY_MAX = 255
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
		case uint64:
			src = make([]byte, 8)
			binary.BigEndian.PutUint64(src, partion.(uint64))
		case []byte:
			src = partion.([]byte)
			buffer = append(buffer, KEY_BYTE_SLICE)
		case string:
			if partion.(string) == "KEY_MAX" {
				buffer = append(buffer, KEY_MAX-1)
			}
		}

		buffer = append(buffer, src...)

		dbKey.KeyPartionList = append(dbKey.KeyPartionList, buffer)
	}

	return proto.Marshal(dbKey)
}
