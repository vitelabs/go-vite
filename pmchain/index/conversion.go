package chain_index

import "encoding/binary"

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

func Uint8ToFixedBytes(height uint64) []byte {
	bytes := make([]byte, 0, 8)
	binary.BigEndian.PutUint64(bytes, height)
	return bytes
}
