package types

import (
	"fmt"
	vcrypto "github.com/vitelabs/go-vite/crypto"
)

const (
	GID_SIZE = 10
)

var (
	PRIVATE_GID  = [GID_SIZE]byte{0}
	SNAPSHOT_GID = [GID_SIZE]byte{1}
	DELEGATE_GID = [GID_SIZE]byte{2}
)

type Gid [GID_SIZE]byte

func CreateGid(data ...[]byte) Gid {
	gid, _ := BytesToGid(vcrypto.Hash(GID_SIZE, data...))
	return gid
}

func BytesToGid(b []byte) (Gid, error) {
	var g Gid
	err := g.SetBytes(b)
	return g, err
}

func (gid *Gid) SetBytes(b []byte) error {
	if length := len(b); length != GID_SIZE {
		return fmt.Errorf("error gid size  %v", length)
	}
	copy(gid[:], b)
	return nil
}
