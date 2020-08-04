package contracts

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/helper"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	abi2 "github.com/vitelabs/go-vite/vm/contracts/abi"
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

func TestName(t *testing.T) {
	addr := types.AddressAsset
	data1 := []byte{26, 219, 85, 114, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 86, 73, 84, 69, 32, 84, 79, 75, 69, 78, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 139, 187, 108, 2, 179, 1, 103, 43, 128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20, 227, 104, 113, 237, 111, 178, 60, 70, 212, 110, 202, 253, 24, 138, 204, 168, 11, 111, 110, 0}
	data2 := []byte{26, 219, 85, 114, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 86, 73, 84, 69, 32, 84, 79, 75, 69, 78, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 139, 187, 117, 191, 10, 28, 85, 119, 193, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20, 227, 104, 113, 237, 111, 178, 60, 70, 212, 110, 202, 253, 24, 138, 204, 168, 11, 111, 110, 0}
	method, _, _ := GetBuiltinContractMethod(addr, data1, 22906)
	t.Log(GetBuiltinContractMethod(addr, data2, 22906))
	methodName := method.(*MethodReIssue).MethodName
	t.Log(methodName)
	param := new(abi2.ParamReIssue)
	abi2.ABIAsset.UnpackMethod(param, "Issue", data1)

	t.Log(param)
	abi2.ABIAsset.UnpackMethod(param, "Issue", data2)
	t.Log(param)

	addr2 := types.HexToAddressPanic("vite_0000000000000000000000000000000000000004d28108e76b")
	data3, err := base64.StdEncoding.DecodeString("zh8npwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGAAAAAAAAAAAAAAABTjaHHtb7I8RtRuyv0YisyoC29uAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIdml0ZS5OTzEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	helper.AssertNil(err)

	method, _, _ = GetBuiltinContractMethod(addr2, data3, 22906)
	t.Log(method)
}
