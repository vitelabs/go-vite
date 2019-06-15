package types

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/vitelabs/go-vite/crypto"
)

const (
	HashSize = 32
)

type Hash [HashSize]byte

var ZERO_HASH = Hash{}

func BytesToHash(b []byte) (Hash, error) {
	var h Hash
	err := h.SetBytes(b)
	return h, err
}

func HexToHash(hexstr string) (Hash, error) {
	if len(hexstr) != 2*HashSize {
		return Hash{}, fmt.Errorf("error hex hash size %v", len(hexstr))
	}
	b, err := hex.DecodeString(hexstr)
	if err != nil {
		return Hash{}, err
	}
	return BytesToHash(b)
}

func HexToHashPanic(hexstr string) Hash {
	h, err := HexToHash(hexstr)
	if err != nil {
		panic(err)
	}
	return h
}

func (h *Hash) SetBytes(b []byte) error {
	if len(b) != HashSize {
		return fmt.Errorf("error hash size %v", len(b))
	}
	copy(h[:], b)
	return nil
}

func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

func (h Hash) Bytes() []byte {
	return h[:]
}

func (h Hash) String() string {
	return h.Hex()
}

func (h Hash) Big() *big.Int {
	return new(big.Int).SetBytes(h[:])
}

func (h Hash) IsZero() bool {
	return h == ZERO_HASH
}

func BigToHash(b *big.Int) (Hash, error) {
	slice := b.Bytes()
	if len(slice) < HashSize {
		padded := make([]byte, HashSize)
		copy(padded[HashSize-len(slice):], slice)
		return BytesToHash(padded)
	} else {
		return BytesToHash(slice)
	}
}

func DataHash(data []byte) Hash {
	h, _ := BytesToHash(crypto.Hash256(data))
	return h
}

func DataListHash(data ...[]byte) Hash {
	h, _ := BytesToHash(crypto.Hash256(data...))
	return h
}

func (h *Hash) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return ErrJsonNotString
	}
	hash, e := HexToHash(string(trimLeftRightQuotation(input)))
	if e != nil {
		return e
	}
	h.SetBytes(hash.Bytes())
	return nil
}

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}
