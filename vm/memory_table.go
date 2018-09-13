package vm

import (
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

// calculates the memory size required for a step
func calcMemSize(off, l *big.Int) *big.Int {
	if l.Sign() == 0 {
		return util.Big0
	}

	return new(big.Int).Add(off, l)
}

func memoryBlake2b(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), stack.back(1))
}

func memoryCallDataCopy(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), stack.back(2))
}

func memoryCodeCopy(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), stack.back(2))
}

func memoryExtCodeCopy(stack *stack) *big.Int {
	return calcMemSize(stack.back(1), stack.back(3))
}

func memoryReturnDataCopy(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), stack.back(2))
}

func memoryMLoad(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), util.Big32)
}

func memoryMStore(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), util.Big32)
}

func memoryMStore8(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), util.Big1)
}

func memoryLog(stack *stack) *big.Int {
	mSize, mStart := stack.back(1), stack.back(0)
	return calcMemSize(mStart, mSize)
}

func memoryDelegateCall(stack *stack) *big.Int {
	x := calcMemSize(stack.back(3), stack.back(4))
	y := calcMemSize(stack.back(1), stack.back(2))
	return util.BigMax(x, y)
}

func memoryReturn(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), stack.back(1))
}

func memoryRevert(stack *stack) *big.Int {
	return calcMemSize(stack.back(0), stack.back(1))
}
