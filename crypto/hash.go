package crypto

import (
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

func Hash256(data ...[]byte) []byte {
	d, _ := blake2b.New256(nil)
	for _, item := range data {
		d.Write(item)
	}
	return d.Sum(nil)
}

func Keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, item := range data {
		d.Write(item)
	}
	return d.Sum(nil)
}

func Hash512(data ...[]byte) []byte {
	d, _ := blake2b.New512(nil)
	for _, item := range data {
		d.Write(item)
	}
	return d.Sum(nil)
}

func Hash(size int, data ...[]byte) []byte {
	d, _ := blake2b.New(size, nil)
	for _, item := range data {
		d.Write(item)
	}
	return d.Sum(nil)
}
