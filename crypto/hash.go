package crypto

import "golang.org/x/crypto/blake2b"

func Hash256(data ...[]byte) []byte {
	d, _ := blake2b.New256(nil)
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
