package vm

import (
	"encoding/hex"
	"math/big"
	"testing"
)

type twoOperandTest struct {
	x        string
	y        string
	expected string
}

func opBenchmark(bench *testing.B, op func(pc *uint64, vm *VM, contract *contract, memory *memory, stack *stack) ([]byte, error), args ...string) {
	vm := &VM{}
	//vm.Debug = true
	c := &contract{intPool: poolOfIntPools.get(), db: NewNoDatabase()}
	stack := newStack()

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
		op(&pc, vm, c, nil, stack)
		stack.pop()
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
	c := &contract{intPool: poolOfIntPools.get(), db: NewNoDatabase()}
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
	c := &contract{intPool: poolOfIntPools.get(), db: NewNoDatabase()}
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
	c := &contract{intPool: poolOfIntPools.get(), db: NewNoDatabase()}
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
	c := &contract{intPool: poolOfIntPools.get(), db: NewNoDatabase()}
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
