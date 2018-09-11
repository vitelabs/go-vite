package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
)

// memoryGasCosts calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.
func memoryGasCost(mem *memory, newMemSize uint64) (uint64, error) {

	if newMemSize == 0 {
		return 0, nil
	}
	// The maximum that will fit in a uint64 is max_word_count - 1
	// anything above that will result in an overflow.
	// Additionally, a newMemSize which results in a
	// newMemSizeWords larger than 0x7ffffffff will cause the square operation
	// to overflow.
	// The constant ç is the highest number that can be used without
	// overflowing the gas calculation
	if newMemSize > 0xffffffffe0 {
		return 0, errGasUintOverflow
	}

	newMemSizeWords := util.ToWordSize(newMemSize)
	newMemSize = newMemSizeWords * 32

	if newMemSize > uint64(mem.len()) {
		square := newMemSizeWords * newMemSizeWords
		linCoef := newMemSizeWords * memoryGas
		quadCoef := square / quadCoeffDiv
		newTotalFee := linCoef + quadCoef

		fee := newTotalFee - mem.lastGasCost
		mem.lastGasCost = newTotalFee

		return fee, nil
	}
	return 0, nil
}

func constGasFunc(gas uint64) gasFunc {
	return func(vm *VM, contrac *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
		return gas, nil
	}
}

func gasExp(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	expByteLen := uint64((stack.back(1).BitLen() + 7) / 8)

	var (
		gas      = expByteLen * expByteGas // no overflow check required. Max is 256 * expByteGas gas
		overflow bool
	)
	if gas, overflow = util.SafeAdd(gas, slowStepGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasBlake2b(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	if gas, overflow = util.SafeAdd(gas, blake2bGas); overflow {
		return 0, errGasUintOverflow
	}

	wordGas, overflow := util.BigUint64(stack.back(1))
	if overflow {
		return 0, errGasUintOverflow
	}
	if wordGas, overflow = util.SafeMul(util.ToWordSize(wordGas), blake2bWordGas); overflow {
		return 0, errGasUintOverflow
	}
	if gas, overflow = util.SafeAdd(gas, wordGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasCallDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = util.SafeAdd(gas, fastestStepGas); overflow {
		return 0, errGasUintOverflow
	}

	words, overflow := util.BigUint64(stack.back(2))
	if overflow {
		return 0, errGasUintOverflow
	}

	if words, overflow = util.SafeMul(util.ToWordSize(words), copyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = util.SafeAdd(gas, words); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = util.SafeAdd(gas, fastestStepGas); overflow {
		return 0, errGasUintOverflow
	}

	wordGas, overflow := util.BigUint64(stack.back(2))
	if overflow {
		return 0, errGasUintOverflow
	}
	if wordGas, overflow = util.SafeMul(util.ToWordSize(wordGas), copyGas); overflow {
		return 0, errGasUintOverflow
	}
	if gas, overflow = util.SafeAdd(gas, wordGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasExtCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = util.SafeAdd(gas, extCodeCopyGas); overflow {
		return 0, errGasUintOverflow
	}

	wordGas, overflow := util.BigUint64(stack.back(3))
	if overflow {
		return 0, errGasUintOverflow
	}

	if wordGas, overflow = util.SafeMul(util.ToWordSize(wordGas), copyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = util.SafeAdd(gas, wordGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasReturnDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = util.SafeAdd(gas, fastestStepGas); overflow {
		return 0, errGasUintOverflow
	}

	words, overflow := util.BigUint64(stack.back(2))
	if overflow {
		return 0, errGasUintOverflow
	}

	if words, overflow = util.SafeMul(util.ToWordSize(words), copyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = util.SafeAdd(gas, words); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasMLoad(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, errGasUintOverflow
	}
	if gas, overflow = util.SafeAdd(gas, fastestStepGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasMStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, errGasUintOverflow
	}
	if gas, overflow = util.SafeAdd(gas, fastestStepGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasMStore8(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, errGasUintOverflow
	}
	if gas, overflow = util.SafeAdd(gas, fastestStepGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasSStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var (
		y          = stack.back(1)
		loc        = stack.back(0)
		locHash, _ = types.BigToHash(loc)
		val        = vm.Db.Storage(c.address, locHash)
	)
	if len(val) == 0 && y.Sign() != 0 {
		return sstoreSetGas, nil
	} else if len(val) == 0 && y.Sign() == 0 {
		c.quotaRefund = c.quotaRefund + sstoreRefundGas
		return sstoreClearGas, nil
	} else {
		return sstoreResetGas, nil
	}
}

func gasPush(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return fastestStepGas, nil
}

func gasDup(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return fastestStepGas, nil
}

func gasSwap(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return fastestStepGas, nil
}

func makeGasLog(n uint64) gasFunc {
	return func(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
		requestedSize, overflow := util.BigUint64(stack.back(1))
		if overflow {
			return 0, errGasUintOverflow
		}

		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, err
		}

		if gas, overflow = util.SafeAdd(gas, logGas); overflow {
			return 0, errGasUintOverflow
		}
		if gas, overflow = util.SafeAdd(gas, n*logTopicGas); overflow {
			return 0, errGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = util.SafeMul(requestedSize, logDataGas); overflow {
			return 0, errGasUintOverflow
		}
		if gas, overflow = util.SafeAdd(gas, memorySizeGas); overflow {
			return 0, errGasUintOverflow
		}
		return gas, nil
	}
}

func gasDelegateCall(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = util.SafeAdd(gas, callGas); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}

func gasReturn(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func gasRevert(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func intrinsicGasCost(data []byte, isCreate bool) (uint64, error) {
	var gas uint64
	if isCreate {
		gas = txContractCreationGas
	} else {
		gas = txGas
	}
	gasData, err := dataGasCost(data)
	if err != nil || util.MaxUint64-gas < gasData {
		return 0, errGasUintOverflow
	}
	return gas + gasData, nil
}

func dataGasCost(data []byte) (uint64, error) {
	var gas uint64
	if len(data) > 0 {
		var nonZeroByteCount uint64
		for _, byteCode := range data {
			if byteCode != 0 {
				nonZeroByteCount++
			}
		}
		if util.MaxUint64/txDataNonZeroGas < nonZeroByteCount {
			return 0, errGasUintOverflow
		}
		gas = nonZeroByteCount * txDataNonZeroGas

		zeroByteCount := uint64(len(data)) - nonZeroByteCount
		if (util.MaxUint64-gas)/txDataZeroGas < zeroByteCount {
			return 0, errGasUintOverflow
		}
		gas += zeroByteCount * txDataZeroGas
	}
	return gas, nil
}
