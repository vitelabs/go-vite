package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"testing"
	"time"
)

type twoOperandTest struct {
	x        string
	y        string
	expected string
}

func opBenchmark(bench *testing.B, op func(pc *uint64, vm *VM, contract *contract, memory *memory, stack *stack) ([]byte, error), args ...string) {
	ts := time.Now()
	vm := &VM{
		globalStatus: NewTestGlobalStatus(100, &ledger.SnapshotBlock{Timestamp: &ts, Height: 1, Hash: types.Hash{}}),
	}
	//vm.Debug = true
	sendCallBlock := &ledger.AccountBlock{
		BlockType:  ledger.BlockTypeSendCall,
		Data:       []byte{},
		Amount:     big.NewInt(10),
		Fee:        big.NewInt(0),
		TokenId:    ledger.ViteTokenId,
		Difficulty: big.NewInt(67108863),
	}
	sendCallBlock.Data, _ = hex.DecodeString("cbf0e4fa000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000002e90edd0000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000001052654973737561626c6520546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000027274000000000000000000000000000000000000000000000000000000000000")
	sendCallBlock.AccountAddress, _ = types.HexToAddress("vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906")
	sendCallBlock.ToAddress, _ = types.HexToAddress("vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68")

	receiveCallBlock := &ledger.AccountBlock{
		BlockType:  ledger.BlockTypeReceive,
		Difficulty: big.NewInt(67108863),
	}
	c := &contract{intPool: poolOfIntPools.get(), db: newNoDatabase(), block: receiveCallBlock, sendBlock: sendCallBlock, jumpdests: make(destinations)}
	code, _ := hex.DecodeString("608060405234801561001057600080fd5b50610141806100206000396000f3fe608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806391a6cb4b14610046575b600080fd5b6100896004803603602081101561005c57600080fd5b81019080803574ffffffffffffffffffffffffffffffffffffffffff16906020019092919050505061008b565b005b8074ffffffffffffffffffffffffffffffffffffffffff164669ffffffffffffffffffff163460405160405180820390838587f1505050508074ffffffffffffffffffffffffffffffffffffffffff167faa65281f5df4b4bd3c71f2ba25905b907205fce0809a816ef8e04b4d496a85bb346040518082815260200191505060405180910390a25056fea165627a7a7230582036a610e43120f537e367e329d2835ba4369e3fe2755c8a32f675fe81f4a971db0029")
	c.setCallCode(sendCallBlock.ToAddress, code)
	c.returnData, _ = hex.DecodeString("000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000002e90edd0000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000001052654973737561626c6520546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000027274000000000000000000000000000000000000000000000000000000000000")
	stack := newStack()
	mem := newMemory()
	mem.resize(1024)

	// convert args
	byteArgs := make([][]byte, len(args))
	for i, arg := range args {
		byteArgs[i], _ = hex.DecodeString(arg)
	}
	pc := uint64(0)
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		for _, arg := range byteArgs {
			a := new(big.Int).SetBytes(arg)
			stack.push(a)
		}
		op(&pc, vm, c, mem, stack)
		for i := stack.len(); i > 0; i-- {
			stack.pop()
		}
	}
	poolOfIntPools.put(c.intPool)
}

func BenchmarkOpAdd64(b *testing.B) {
	x := "ffffffff"
	y := "fd37f3e2bba2c4f"

	opBenchmark(b, opAdd, x, y)
}

func BenchmarkOpAdd128(b *testing.B) {
	x := "ffffffffffffffff"
	y := "f5470b43c6549b016288e9a65629687"

	opBenchmark(b, opAdd, x, y)
}

func BenchmarkOpAdd256(b *testing.B) {
	x := "0802431afcbce1fc194c9eaa417b2fb67dc75a95db0bc7ec6b1c8af11df6a1da9"
	y := "a1f5aac137876480252e5dcac62c354ec0d42b76b0642b6181ed099849ea1d57"

	opBenchmark(b, opAdd, x, y)
}

func BenchmarkOpSub64(b *testing.B) {
	x := "51022b6317003a9d"
	y := "a20456c62e00753a"

	opBenchmark(b, opSub, x, y)
}

func BenchmarkOpSub128(b *testing.B) {
	x := "4dde30faaacdc14d00327aac314e915d"
	y := "9bbc61f5559b829a0064f558629d22ba"

	opBenchmark(b, opSub, x, y)
}

func BenchmarkOpSub256(b *testing.B) {
	x := "4bfcd8bb2ac462735b48a17580690283980aa2d679f091c64364594df113ea37"
	y := "97f9b1765588c4e6b69142eb00d20507301545acf3e1238c86c8b29be227d46e"

	opBenchmark(b, opSub, x, y)
}

func BenchmarkOpMul(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opMul, x, y)
}

func BenchmarkOpDiv256(b *testing.B) {
	x := "ff3f9014f20db29ae04af2c2d265de17"
	y := "fe7fb0d1f59dfe9492ffbf73683fd1e870eec79504c60144cc7f5fc2bad1e611"
	opBenchmark(b, opDiv, x, y)
}

func BenchmarkOpDiv128(b *testing.B) {
	x := "fdedc7f10142ff97"
	y := "fbdfda0e2ce356173d1993d5f70a2b11"
	opBenchmark(b, opDiv, x, y)
}

func BenchmarkOpDiv64(b *testing.B) {
	x := "fcb34eb3"
	y := "f97180878e839129"
	opBenchmark(b, opDiv, x, y)
}

func BenchmarkOpSdiv(b *testing.B) {
	x := "ff3f9014f20db29ae04af2c2d265de17"
	y := "fe7fb0d1f59dfe9492ffbf73683fd1e870eec79504c60144cc7f5fc2bad1e611"

	opBenchmark(b, opSdiv, x, y)
}

func BenchmarkOpMod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opMod, x, y)
}

func BenchmarkOpSmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSmod, x, y)
}

func BenchmarkOpExp(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opExp, x, y)
}

func BenchmarkOpSignExtend(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSignExtend, x, y)
}

func BenchmarkOpLt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opLt, x, y)
}

func BenchmarkOpGt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opGt, x, y)
}

func BenchmarkOpSlt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSlt, x, y)
}

func BenchmarkOpSgt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSgt, x, y)
}

func BenchmarkOpEq(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opEq, x, y)
}
func BenchmarkOpEq2(b *testing.B) {
	x := "FBCDEF090807060504030201ffffffffFBCDEF090807060504030201ffffffff"
	y := "FBCDEF090807060504030201ffffffffFBCDEF090807060504030201fffffffe"
	opBenchmark(b, opEq, x, y)
}
func BenchmarkOpAnd(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opAnd, x, y)
}

func BenchmarkOpOr(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opOr, x, y)
}

func BenchmarkOpXor(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opXor, x, y)
}

func BenchmarkOpByte(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opByte, x, y)
}

func BenchmarkOpAddmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	z := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opAddmod, x, y, z)
}

func BenchmarkOpMulmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	z := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opMulmod, x, y, z)
}

func BenchmarkOpSHL(b *testing.B) {
	x := "FBCDEF090807060504030201ffffffffFBCDEF090807060504030201ffffffff"
	y := "ff"

	opBenchmark(b, opSHL, x, y)
}
func BenchmarkOpSHR(b *testing.B) {
	x := "FBCDEF090807060504030201ffffffffFBCDEF090807060504030201ffffffff"
	y := "ff"

	opBenchmark(b, opSHR, x, y)
}
func BenchmarkOpSAR(b *testing.B) {
	x := "FBCDEF090807060504030201ffffffffFBCDEF090807060504030201ffffffff"
	y := "ff"

	opBenchmark(b, opSAR, x, y)
}
func BenchmarkOpIsZero(b *testing.B) {
	x := "FBCDEF090807060504030201ffffffffFBCDEF090807060504030201ffffffff"
	opBenchmark(b, opIszero, x)
}

func TestByteOp(t *testing.T) {
	tests := []struct {
		v        string
		th       uint64
		expected *big.Int
	}{
		{"ABCDEF0908070605040302010000000000000000000000000000000000000000", 0, big.NewInt(0xAB)},
		{"ABCDEF0908070605040302010000000000000000000000000000000000000000", 1, big.NewInt(0xCD)},
		{"00CDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff", 0, big.NewInt(0x00)},
		{"00CDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff", 1, big.NewInt(0xCD)},
		{"0000000000000000000000000000000000000000000000000000000000102030", 31, big.NewInt(0x30)},
		{"0000000000000000000000000000000000000000000000000000000000102030", 30, big.NewInt(0x20)},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 32, big.NewInt(0x0)},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0xFFFFFFFFFFFFFFFF, big.NewInt(0x0)},
	}
	vm := &VM{}
	//vm.Debug = true
	c := &contract{intPool: poolOfIntPools.get(), db: newNoDatabase()}
	stack := newStack()
	pc := uint64(0)
	for _, test := range tests {
		v, _ := hex.DecodeString(test.v)
		val := new(big.Int).SetBytes(v)
		th := new(big.Int).SetUint64(test.th)
		stack.push(val)
		stack.push(th)
		opByte(&pc, vm, c, nil, stack)
		actual := stack.pop()
		if actual.Cmp(test.expected) != 0 {
			t.Fatalf("Expected  [%v] %v:th byte to be %v, was %v.", test.v, test.th, test.expected, actual)
		}
	}
	poolOfIntPools.put(c.intPool)
}

func testTwoOperandOp(t *testing.T, tests []twoOperandTest, opFn func(pc *uint64, vm *VM, contract *contract, memory *memory, stack *stack) ([]byte, error)) {
	vm := &VM{}
	//vm.Debug = true
	c := &contract{intPool: poolOfIntPools.get(), db: newNoDatabase()}
	stack := newStack()
	pc := uint64(0)
	for i, test := range tests {
		x, _ := hex.DecodeString(test.x)
		y, _ := hex.DecodeString(test.y)
		expected, _ := hex.DecodeString(test.expected)
		expectedInt := new(big.Int).SetBytes(expected)
		stack.push(new(big.Int).SetBytes(x))
		stack.push(new(big.Int).SetBytes(y))
		opFn(&pc, vm, c, nil, stack)
		actual := stack.pop()
		if actual.Cmp(expectedInt) != 0 {
			t.Errorf("Testcase %d, expected  %v, got %v", i, expectedInt, actual)
		}
	}
	poolOfIntPools.put(c.intPool)
}

func TestSHL(t *testing.T) {
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "00", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "01", "0000000000000000000000000000000000000000000000000000000000000002"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "ff", "8000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "0101", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "00", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "8000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000000", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
	}
	testTwoOperandOp(t, tests, opSHL)
}

func TestSHR(t *testing.T) {
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "00", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "01", "4000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "ff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0101", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "00", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000000", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
	}
	testTwoOperandOp(t, tests, opSHR)
}

func TestSAR(t *testing.T) {
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "00", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "01", "c000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "ff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0100", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0101", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "00", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0000000000000000000000000000000000000000000000000000000000000000", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"4000000000000000000000000000000000000000000000000000000000000000", "fe", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "f8", "000000000000000000000000000000000000000000000000000000000000007f"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "fe", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
	}

	testTwoOperandOp(t, tests, opSAR)
}

func TestSGT(t *testing.T) {
	tests := []twoOperandTest{

		{"0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "0000000000000000000000000000000000000000000000000000000000000000"},
	}
	testTwoOperandOp(t, tests, opSgt)
}

func TestSLT(t *testing.T) {
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "0000000000000000000000000000000000000000000000000000000000000001"},
	}
	testTwoOperandOp(t, tests, opSlt)
}

func TestOpMstore(t *testing.T) {
	vm := &VM{}
	//vm.Debug = true
	c := &contract{intPool: poolOfIntPools.get(), db: newNoDatabase()}
	stack := newStack()
	mem := newMemory()
	mem.resize(64)
	pc := uint64(0)
	v := "abcdef00000000000000abba000000000deaf000000c0de00100000000133700"
	vbytes, _ := hex.DecodeString("abcdef00000000000000abba000000000deaf000000c0de00100000000133700")
	stack.push(new(big.Int).SetBytes(vbytes))
	stack.push(big.NewInt(0))
	opMstore(&pc, vm, c, mem, stack)
	if got := hex.EncodeToString(mem.get(0, 32)); got != v {
		t.Fatalf("Mstore fail, got %v, expected %v", got, v)
	}
	stack.push(big.NewInt(0x1))
	stack.push(big.NewInt(0))
	opMstore(&pc, vm, c, mem, stack)
	if hex.EncodeToString(mem.get(0, 32)) != "0000000000000000000000000000000000000000000000000000000000000001" {
		t.Fatalf("Mstore failed to overwrite previous value")
	}
	poolOfIntPools.put(c.intPool)
}

func BenchmarkOpMstore(bench *testing.B) {
	vm := &VM{}
	//vm.Debug = true
	c := &contract{intPool: poolOfIntPools.get(), db: newNoDatabase()}
	stack := newStack()
	mem := newMemory()
	mem.resize(64)
	pc := uint64(0)
	memStart := big.NewInt(0)
	value := big.NewInt(0x1337)

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		stack.push(value)
		stack.push(memStart)
		opMstore(&pc, vm, c, mem, stack)
	}
	poolOfIntPools.put(c.intPool)
}
func BenchmarkOpMstore8(bench *testing.B) {
	vm := &VM{}
	//vm.Debug = true
	c := &contract{intPool: poolOfIntPools.get(), db: newNoDatabase()}
	stack := newStack()
	mem := newMemory()
	mem.resize(64)
	pc := uint64(0)
	memStart := big.NewInt(0)
	value := big.NewInt(0x13)

	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		stack.push(value)
		stack.push(memStart)
		opMstore8(&pc, vm, c, mem, stack)
	}
	poolOfIntPools.put(c.intPool)
}

func BenchmarkRandom(b *testing.B) {
	opBenchmark(b, opRandom)
}

func BenchmarkSeed(b *testing.B) {
	opBenchmark(b, opSeed)
}
func BenchmarkNot(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	opBenchmark(b, opNot, x)
}
func BenchmarkAddress(b *testing.B) {
	opBenchmark(b, opAddress)
}
func BenchmarkCaller(b *testing.B) {
	opBenchmark(b, opCaller)
}
func BenchmarkCallValue(b *testing.B) {
	opBenchmark(b, opCallValue)
}
func BenchmarkCallDataLoad(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, opCallDataLoad, x)
}
func BenchmarkCallDataSize(b *testing.B) {
	opBenchmark(b, opCallDataSize)
}
func BenchmarkCallDataCopy(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "0000000000000000000000000000000000000000000000000000000000000000"
	z := "000000000000000000000000000000000000000000000000000000000000000a"
	opBenchmark(b, opCallDataCopy, x, y, z)
}
func BenchmarkReturnDataSize(b *testing.B) {
	opBenchmark(b, opReturnDataSize)
}
func BenchmarkReturnDataCopy(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "0000000000000000000000000000000000000000000000000000000000000000"
	z := "0000000000000000000000000000000000000000000000000000000000000020"
	opBenchmark(b, opReturnDataCopy, x, y, z)
}
func BenchmarkCodeSize(b *testing.B) {
	opBenchmark(b, opCodeSize)
}
func BenchmarkCodeCopy(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "0000000000000000000000000000000000000000000000000000000000000000"
	z := "000000000000000000000000000000000000000000000000000000000000000a"
	opBenchmark(b, opCodeCopy, x, y, z)
}
func BenchmarkBlake2b(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "000000000000000000000000000000000000000000000000000000000000000a"
	opBenchmark(b, opBlake2b, x, y)
}
func BenchmarkTimestamp(b *testing.B) {
	opBenchmark(b, opTimestamp)
}
func BenchmarkHeight(b *testing.B) {
	opBenchmark(b, opHeight)
}
func BenchmarkTokenId(b *testing.B) {
	opBenchmark(b, opTokenID)
}
func BenchmarkAccountHeight(b *testing.B) {
	opBenchmark(b, opAccountHeight)
}
func BenchmarkPrevHash(b *testing.B) {
	opBenchmark(b, opAccountHash)
}
func BenchmarkFromHash(b *testing.B) {
	opBenchmark(b, opFromHash)
}
func BenchmarkPop(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, opPop, x)
}
func BenchmarkJump(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000010"
	opBenchmark(b, opJump, x)
}
func BenchmarkJumpi(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000001"
	y := "0000000000000000000000000000000000000000000000000000000000000010"
	opBenchmark(b, opJumpi, x, y)
}
func BenchmarkPc(b *testing.B) {
	opBenchmark(b, opPc)
}
func BenchmarkMSize(b *testing.B) {
	opBenchmark(b, opMsize)
}
func BenchmarkJumpDest(b *testing.B) {
	opBenchmark(b, opJumpdest)
}
func BenchmarkPush(b *testing.B) {
	opBenchmark(b, makePush(32, 32))
}
func BenchmarkDup(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, makeDup(1), x)
}
func BenchmarkSwap(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, makeSwap(1), x, y)
}
func BenchmarkReturn(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "0000000000000000000000000000000000000000000000000000000000000020"
	opBenchmark(b, opReturn, x, y)
}
func BenchmarkRevert(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	y := "0000000000000000000000000000000000000000000000000000000000000020"
	opBenchmark(b, opRevert, x, y)
}
func BenchmarkMLoad(b *testing.B) {
	x := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, opMload, x)
}
func BenchmarkLog0(b *testing.B) {
	start := "0000000000000000000000000000000000000000000000000000000000000000"
	size := "0000000000000000000000000000000000000000000000000000000000000020"
	opBenchmark(b, makeLog(0), start, size)
}
func BenchmarkLog1(b *testing.B) {
	start := "0000000000000000000000000000000000000000000000000000000000000000"
	size := "0000000000000000000000000000000000000000000000000000000000000020"
	topic1 := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, makeLog(1), start, size, topic1)
}
func BenchmarkLog2(b *testing.B) {
	start := "0000000000000000000000000000000000000000000000000000000000000000"
	size := "0000000000000000000000000000000000000000000000000000000000000020"
	topic1 := "0000000000000000000000000000000000000000000000000000000000000000"
	topic2 := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, makeLog(2), start, size, topic1, topic2)
}
func BenchmarkLog3(b *testing.B) {
	start := "0000000000000000000000000000000000000000000000000000000000000000"
	size := "0000000000000000000000000000000000000000000000000000000000000020"
	topic1 := "0000000000000000000000000000000000000000000000000000000000000000"
	topic2 := "0000000000000000000000000000000000000000000000000000000000000000"
	topic3 := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, makeLog(3), start, size, topic1, topic2, topic3)
}
func BenchmarkLog4(b *testing.B) {
	start := "0000000000000000000000000000000000000000000000000000000000000000"
	size := "0000000000000000000000000000000000000000000000000000000000000020"
	topic1 := "0000000000000000000000000000000000000000000000000000000000000000"
	topic2 := "0000000000000000000000000000000000000000000000000000000000000000"
	topic3 := "0000000000000000000000000000000000000000000000000000000000000000"
	topic4 := "0000000000000000000000000000000000000000000000000000000000000000"
	opBenchmark(b, makeLog(4), start, size, topic1, topic2, topic3, topic4)
}
func BenchmarkCall(b *testing.B) {
	addr := "0000000000000000000000e41be57d38c796984952fad618a9bc91637329b500"
	tokenID := "000000000000000000000000000000000000000000001949754100b9b491e7d3"
	amount := "0000000000000000000000000000000000000000000000000000000000000010"
	offset := "0000000000000000000000000000000000000000000000000000000000000000"
	size := "0000000000000000000000000000000000000000000000000000000000000020"
	opBenchmark(b, opCall, addr, tokenID, amount, offset, size)
}
func BenchmarkSstore(b *testing.B) {
	vm := &VM{}
	addr, _, _ := types.CreateAddress()
	c := &contract{intPool: poolOfIntPools.get(), db: vm_db.NewGenesisVmDB(&addr)}
	stack := newStack()
	mem := newMemory()
	mem.resize(64)
	pc := uint64(0)
	loc := big.NewInt(0)
	value := big.NewInt(0x13)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stack.push(value)
		stack.push(loc)
		opSStore(&pc, vm, c, mem, stack)
	}
	poolOfIntPools.put(c.intPool)
}
