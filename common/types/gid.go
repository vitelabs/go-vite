package types

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
)

const (
	GidSize        = 10
	hexGidIdLength = 2 * GidSize
)

var (
	PRIVATE_GID  = Gid{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	SNAPSHOT_GID = Gid{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	DELEGATE_GID = Gid{0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
)

type Gid [GidSize]byte

func DataToGid(data ...[]byte) Gid {
	gid, _ := BytesToGid(crypto.Hash(GidSize, data...))
	return gid
}
func HexToGid(hexStr string) (Gid, error) {
	if len(hexStr) != hexGidIdLength {
		return Gid{}, fmt.Errorf("Not valid hex gid")
	}
	gid, err := hex.DecodeString(hexStr)
	if err != nil {
		return Gid{}, fmt.Errorf("Not valid hex gid")
	}
	return BytesToGid(gid)
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
func (gid Gid) Hex() string {
	return hex.EncodeToString(gid[:])
}
func (gid Gid) String() string {
	return gid.Hex()
}
func (gid *Gid) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return ErrJsonNotString
	}
	g, e := HexToGid(string(trimLeftRightQuotation(input)))
	if e != nil {
		return e
	}
	gid.SetBytes(g.Bytes())
	return nil
}

func (gid Gid) MarshalText() ([]byte, error) {
	return []byte(gid.String()), nil
}
