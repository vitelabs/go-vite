package chain_index

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"testing"
)

func uint64ToBytes(data uint64) []byte {
	src := make([]byte, 8)
	binary.BigEndian.PutUint64(src, data)
	return src
}

func EncodeKey(len int, prefix byte, partList ...interface{}) ([]byte, error) {
	buffer := make([]byte, 0, len)
	for _, part := range partList {
		var src []byte
		switch part.(type) {
		case uint64:
			src = uint64ToBytes(part.(uint64))
		case []byte:
			src = part.([]byte)
		default:
			return nil, nil
		}

		buffer = append(buffer, src...)
	}

	return buffer, nil
}

func BmHash1(b *testing.B, hashBytes []byte, hashBytes2 []byte) {
	for i := 0; i < b.N; i++ {
		EncodeKey(1+types.HashSize+types.HashSize, AccountBlockHashKeyPrefix, hashBytes, hashBytes2)
	}
}

func BmHash2(b *testing.B, hashBytes []byte, hashBytes2 []byte) {
	for i := 0; i < b.N; i++ {
		key := make([]byte, 0, 1+types.HashSize+types.HashSize)

		key = append(append(append(key, AccountBlockHashKeyPrefix), hashBytes...), hashBytes2...)
	}
}

func BenchmarkIndexDB_IsAccountBlockExisted(b *testing.B) {

	hash := crypto.Hash256([]byte("This is a hash"))
	b.Run("hash1", func(b *testing.B) {
		BmHash1(b, hash, hash)
	})

	b.Run("hash2", func(b *testing.B) {
		BmHash2(b, hash, hash)
	})

}
