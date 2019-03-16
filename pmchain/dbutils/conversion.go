package chain_dbutils

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func SerializeAccountIdHeight(accountId, height uint64) []byte {
	return nil
}

func DeserializeAccountIdHeight(accountId, height uint64) []byte {
	return nil
}

func SerializeAccountId(accountId uint64) []byte {
	return nil
}

func DeserializeAccountId(buf []byte) uint64 {
	return 0
}

func SerializeHeight(height uint64) []byte {
	return nil
}

func SerializeLocation(location *chain_block.Location) []byte {
	fileIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(fileIdBytes, location.FileId())

	offsetBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(offsetBytes, location.Offset())

	return append(fileIdBytes, offsetBytes...)
}

func DeserializeLocation(bytes []byte) *chain_block.Location {
	return chain_block.NewLocation(FixedBytesToUint64(bytes[:8]), binary.BigEndian.Uint32(bytes[8:]))
}

func Uint64ToFixedBytes(height uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, height)
	return bytes
}

func FixedBytesToUint64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}
