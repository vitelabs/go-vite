package contracts

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"testing"
)

func TestABIContract_MethodById(t *testing.T) {
	for addr, c := range simpleContracts {
		printEvent(addr, c.abi)
	}
}

func printEvent(addr types.Address, contract abi.ABIContract) {
	fmt.Printf("%v \n", addr.String())
	for _, m := range contract.Methods {
		fmt.Printf("%v\t%v\n", m.Name, hex.EncodeToString(m.Id()))
	}
}
