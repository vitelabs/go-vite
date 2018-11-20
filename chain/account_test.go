package chain

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/vm/contracts/abi"

	"github.com/vitelabs/go-vite/common/types"
)

func TestChain_GetAccount(t *testing.T) {
	chainInstance := getChainInstance()
	addr, _ := types.HexToAddress("vite_5acd0b2ef651bdc0c586aafe7a780103f45ac532cd886eb859")
	account, _ := chainInstance.GetAccount(&addr)
	fmt.Printf("%+v\n", account)
}

func TestAccountType(t *testing.T) {
	chainInstance := getChainInstance()

	addr, _ := types.HexToAddress("vite_00000000000000000000000000000000000000056ad6d26692")
	code, _ := chainInstance.AccountType(&addr)
	fmt.Println(code)
}

func TestAccountBalance(t *testing.T) {
	chainInstance := getChainInstance()

	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000001c9e9f25417")
	for i := uint64(2); i <= 12; i++ {
		block, e := chainInstance.GetAccountBlockByHeight(&addr, i)
		if e != nil {
			panic(e)
		}
		if block.BlockType == 2 {
			println("send", strconv.FormatUint(block.Height, 10))
			continue
		}
		printParams(chainInstance, block)
	}

	//printParam2(chainInstance, "3d6b153c65945ef7f21fe459aa16f29c5dbd173d2127a1ceef43255bd3358532")
	//printParam3(chainInstance, "3d6b153c65945ef7f21fe459aa16f29c5dbd173d2127a1ceef43255bd3358532") // 5  update
	//printParam(chainInstance, "1ff817db93c0a311ba974c027fea119c401f504a38768afe9ecf1ba351919bd5")  // 2
	//printParam(chainInstance, "3c79ab2df58d1a9ff8a3452bc99f3b473b64fb50c9c1f39f4db7689ee50f036d")  // 7 err

	//printParam(block)
	//method, e := abi.ABIRegister.MethodById(block.Data[0:4])
	//if e != nil {
	//	panic(e)
	//}
	//
	//t.Log(method.Name)
	//fmt.Println(code)

	//hash, _ := types.HexToHash("f7e4c8516e5e73b50d3e56abb30b0edaf8d871e08810d920d87aee27d6336436")
	//block, e := chainInstance.GetAccountBlockByHash(&hash)
	//if e != nil {
	//	panic(e)
	//}
	////block.Data[:(len(block.Data) - 1)]
	//num := big.NewInt(0).SetBytes(block.Data[(len(block.Data) - 1):])
	//println(num.String())
}
func printParams(c Chain, block *ledger.AccountBlock) {
	byHash, e := c.GetAccountBlockByHash(&block.FromBlockHash)
	if e != nil {
		panic(e)
	}
	if byHash == nil {
		panic(block.FromBlockHash.String())
	}
	printParam(c, byHash)
	println()
}
func printParam(c Chain, block *ledger.AccountBlock) {
	method, e := abi.ABIRegister.MethodById(block.Data[0:4])
	if e != nil {
		panic(e)
	}
	switch method.Name {
	case "Register":
		printRegisterParams(block.Data)
		print("   " + block.Amount.String())
		return
	case "UpdateRegistration":
		printUpdateParams(block.Data)
		print("   " + block.Amount.String())
		return
	}

	panic(method.Name)
}
func printUpdateParams(bytes []byte) {
	param := new(abi.ParamRegister)
	if err := abi.ABIRegister.UnpackMethod(param, abi.MethodNameUpdateRegistration, bytes); err != nil {
		panic(err)
	}
	fmt.Printf("update:%+v", param)
}
func printRegisterParams(bytes []byte) {
	param := new(abi.ParamRegister)
	if err := abi.ABIRegister.UnpackMethod(param, abi.MethodNameRegister, bytes); err != nil {
		panic(err)
	}
	fmt.Printf("register:%+v", param)
}
func printParam2(c Chain, hs string) {
	hash, _ := types.HexToHash(hs)
	block, e := c.GetAccountBlockByHash(&hash)
	if e != nil {
		panic(e)
	}

	method, e := abi.ABIRegister.MethodById(block.Data[0:4])
	if e != nil {
		panic(e)
	}
	fmt.Println(method.Name)

	//param := new(abi.ParamRegister)
	//if err := abi.ABIRegister.UnpackMethod(param, abi.MethodNameRegister, block.Data); err != nil {
	//	panic(err)
	//}
	//
	//fmt.Printf("%+v\n", param)
}

func printParam3(c Chain, hs string) {
	hash, _ := types.HexToHash(hs)
	block, e := c.GetAccountBlockByHash(&hash)
	if e != nil {
		panic(e)
	}

	param := new(abi.ParamRegister)
	if err := abi.ABIRegister.UnpackMethod(param, abi.MethodNameUpdateRegistration, block.Data); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", param)
}
