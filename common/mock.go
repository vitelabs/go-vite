package common

import (
	"encoding/hex"
	"fmt"

	"github.com/vitelabs/go-vite/common/types"
)

func MockAddress(i int) types.Address {
	base := "00000000000000000fff" + fmt.Sprintf("%0*d", 6, i) + "fff00000000000"
	bytes, _ := hex.DecodeString(base)
	addresses, _ := types.BytesToAddress(bytes)
	return addresses
}

func MockHash(i int) types.Hash {
	base := "0000000000000000000000000000fff" + fmt.Sprintf("%0*d", 6, i) + "fff000000000000000000000000"
	bytes, err := hex.DecodeString(base)
	if err != nil {
		panic(err)
	}
	addresses, err := types.BytesToHash(bytes)
	if err != nil {
		panic(err)
	}
	return addresses
}

func MockHashBy(i1 int, i int) types.Hash {
	base := fmt.Sprintf("%0*d", 6, i1) + "0000000000000000000000fff" + fmt.Sprintf("%0*d", 6, i) + "fff000000000000000000000000"
	bytes, err := hex.DecodeString(base)
	if err != nil {
		panic(err)
	}
	addresses, err := types.BytesToHash(bytes)
	if err != nil {
		panic(err)
	}
	return addresses
}
