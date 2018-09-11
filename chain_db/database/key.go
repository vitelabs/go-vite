package database

import (
	"encoding/binary"
	"errors"
)

func EncodeKey(prefix byte, partList ...interface{}) ([]byte, error) {
	var buffer = []byte{prefix}
	for _, part := range partList {
		var src []byte
		switch part.(type) {
		case uint64:
			src = make([]byte, 8)
			binary.BigEndian.PutUint64(src, part.(uint64))
		case []byte:
			src = part.([]byte)
		default:
			return nil, errors.New("Key contains of uint64 and []byte.")
		}

		buffer = append(buffer, src...)
	}

	return buffer, nil
}
