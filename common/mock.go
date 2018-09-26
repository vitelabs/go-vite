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
