package types

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

const (
	HashSize = 32
)

type Hash [HashSize]byte

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

func BigToHash(b *big.Int) (Hash, error) {
	return BytesToHash(b.Bytes())
}

func DataHash(data []byte) Hash {
	h, _ := BytesToHash(crypto.Hash256(data))
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