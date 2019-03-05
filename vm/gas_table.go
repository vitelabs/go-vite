package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
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
		return 0, util.ErrGasUintOverflow
	}

	newMemSizeWords := helper.ToWordSize(newMemSize)
	newMemSize = newMemSizeWords * helper.WordSize

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
	if gas, overflow = helper.SafeAdd(gas, slowStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasBlake2b(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	if gas, overflow = helper.SafeAdd(gas, blake2bGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(1))
	if overflow {
		return 0, util.ErrGasUintOverflow
	}
	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), blake2bWordGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasCallDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	words, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, util.ErrGasUintOverflow
	}

	if words, overflow = helper.SafeMul(helper.ToWordSize(words), copyGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, words); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, util.ErrGasUintOverflow
	}
	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), copyGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasExtCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, extCodeCopyGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(3))
	if overflow {
		return 0, util.ErrGasUintOverflow
	}

	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), copyGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasReturnDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	words, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, util.ErrGasUintOverflow
	}

	if words, overflow = helper.SafeMul(helper.ToWordSize(words), copyGas); overflow {
		return 0, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, words); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasMLoad(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasMStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasMStore8(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasSStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	var (
		newValue     = stack.back(1)
		loc          = stack.back(0)
		locHash, _   = types.BigToHash(loc)
		currentValue = c.db.GetStorage(&c.block.AccountAddress, locHash.Bytes())
	)
	if bytes.Equal(currentValue, newValue.Bytes()) {
		// no change, charge 200
		return sstoreNoopGas, nil
	}
	originalValue := c.db.GetOriginalStorage(locHash.Bytes())
	if bytes.Equal(originalValue, currentValue) {
		if len(originalValue) == 0 {
			// zero value to non-zero value, charge 20000
			return sstoreInitGas, nil
		}
		if newValue.Sign() == 0 {
			// non-zero value to zero value, charge 5000 with 15000 refund
			c.quotaRefund = c.quotaRefund + sstoreClearRefundGas
		}
		// non-zero value to non-zero value, charge 5000
		return sstoreCleanGas, nil
	}
	// value changed again, charge 200
	if len(originalValue) > 0 {
		if len(currentValue) == 0 {
			// non-zero value to zero value to non-zero value, withdraw 15000 refund
			c.quotaRefund = c.quotaRefund - sstoreClearRefundGas
		} else if newValue.Sign() == 0 {
			// non-zero value to non-zero value to zero value,
			c.quotaRefund = c.quotaRefund + sstoreClearRefundGas
		}
	}
	if bytes.Equal(originalValue, newValue.Bytes()) {
		if len(originalValue) == 0 {
			// zero value to non-zero value to zero value, 19800 refund
			c.quotaRefund = c.quotaRefund + sstoreResetClearRefundGas
		} else {
			// non-zero value a to non-zero value b to non-zero value a,4800 refund for first sstore
			c.quotaRefund = c.quotaRefund + sstoreResetRefundGas
		}
	}
	return sstoreDirtyGas, nil
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
		requestedSize, overflow := helper.BigUint64(stack.back(1))
		if overflow {
			return 0, util.ErrGasUintOverflow
		}

		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, err
		}

		if gas, overflow = helper.SafeAdd(gas, logGas); overflow {
			return 0, util.ErrGasUintOverflow
		}
		if gas, overflow = helper.SafeAdd(gas, n*logTopicGas); overflow {
			return 0, util.ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = helper.SafeMul(requestedSize, logDataGas); overflow {
			return 0, util.ErrGasUintOverflow
		}
		if gas, overflow = helper.SafeAdd(gas, memorySizeGas); overflow {
			return 0, util.ErrGasUintOverflow
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
	if gas, overflow = helper.SafeAdd(gas, callGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasCall(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, callGas); overflow {
		return 0, util.ErrGasUintOverflow
	}
	return gas, nil
}

func gasReturn(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func gasRevert(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}
