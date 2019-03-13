package dbutils

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
		case uint64:
			src = uint64ToBytes(part.(uint64))
		case []byte:
			src = part.([]byte)
		default:
			return nil, errors.New("key must contains of only uint64 and []byte")
		}

		buffer = append(buffer, src...)
	}

	return buffer, nil
}
