package types

import (
	"fmt"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

const GidSize = 10

type Gid [GidSize]byte

func DataToGid(data ...[]byte) Gid {
	gid, _ := BytesToGid(crypto.Hash(10, data...))
	return gid
}

func BigToGid(data *big.Int) (Gid, error) {
	slice := data.Bytes()
	if len(slice) < GidSize {
		padded := make([]byte, GidSize)
		copy(padded[GidSize-len(slice):], slice)
		return BytesToGid(padded)
	} else {
		return BytesToGid(slice)
	}
}
func BytesToGid(b []byte) (Gid, error) {
	var gid Gid
	err := gid.SetBytes(b)
	return gid, err
}
func (gid *Gid) SetBytes(b []byte) error {
	if len(b) != GidSize {
		return fmt.Errorf("error gid size %v", len(b))
	}
	copy(gid[:], b)
	return nil
}
func (gid *Gid) Bytes() []byte {
	return gid[:]
}
