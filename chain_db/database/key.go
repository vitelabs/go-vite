package database

import (
	"encoding/binary"
	"errors"
)

func uint64ToBytes(data uint64) []byte {
	src := make([]byte, 8)
	binary.BigEndian.PutUint64(src, data)
	return src
}

func EncodeKey(prefix byte, partList ...interface{}) ([]byte, error) {
	var buffer = []byte{prefix}
	for _, part := range partList {
		var src []byte
		switch part.(type) {
		case int:
			src = uint64ToBytes(uint64(part.(int)))
		case int32:
			src = uint64ToBytes(uint64(part.(int32)))
		case uint32:
			src = uint64ToBytes(uint64(part.(uint32)))
		case int64:
			src = uint64ToBytes(uint64(part.(int64)))
		case uint64:
			src = uint64ToBytes(part.(uint64))
		case []byte:
			src = part.([]byte)
		default:
			return nil, errors.New("Key contains of uint64 and []byte.")
		}

		buffer = append(buffer, src...)
	}

	return buffer, nil
}
