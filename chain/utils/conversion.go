package chain_utils

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/chain/file_manager"
)

func SerializeLocation(location *chain_file_manager.Location) []byte {
	bytes := make([]byte, 12)
	binary.BigEndian.PutUint64(bytes, location.FileId)

	binary.BigEndian.PutUint32(bytes[8:], uint32(location.Offset))

	return bytes
}

func DeserializeLocation(bytes []byte) *chain_file_manager.Location {
	return chain_file_manager.NewLocation(BytesToUint64(bytes[:8]), int64(binary.BigEndian.Uint32(bytes[8:])))
}

func Uint64ToBytes(height uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, height)
	return bytes
}

func BytesToUint64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}
