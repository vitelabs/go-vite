package vm

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/helper"
	"sync/atomic"
)

type interpreter struct {
	instructionSet [256]operation
}

func newInterpreter() *interpreter {
	return &interpreter{simpleInstructionSet}
}

func (i *interpreter) Run(vm *VM, c *contract) (ret []byte, err error) {
	vm.returnData = nil

	var (
		op   opCode
		mem  = newMemory()
		st   = newStack()
		pc   = uint64(0)
		cost uint64
	)

	for atomic.LoadInt32(&vm.abort) == 0 {
		currentPc := pc
		op = c.getOp(pc)
		operation := i.instructionSet[op]

		if !operation.valid {
			return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
		}

		if err := operation.validateStack(st); err != nil {
			return nil, err
		}

		var memorySize uint64
		if operation.memorySize != nil {
			memSize, overflow := helper.BigUint64(operation.memorySize(st))
			if overflow {
				return nil, errGasUintOverflow
			}
			if memorySize, overflow = helper.SafeMul(helper.ToWordSize(memSize), helper.WordSize); overflow {
				return nil, errGasUintOverflow
			}
		}

		cost, err = operation.gasCost(vm, c, st, mem, memorySize)
		if err != nil {
			return nil, err
		}
		c.quotaLeft, err = useQuota(c.quotaLeft, cost)
		if err != nil {
			return nil, err
		}

		if memorySize > 0 {
			mem.resize(memorySize)
		}

		res, err := operation.execute(&pc, vm, c, mem, st)

		if vm.Debug {
			logger.Info("current code", "code", hex.EncodeToString(c.code[currentPc:]))
			fmt.Printf("code: %v \n", hex.EncodeToString(c.code[currentPc:]))
			fmt.Printf("op: %v, pc: %v\nstack: [%v]\nmemory: [%v]\nquotaLeft: %v, quotaRefund: %v\n", opCodeToString[op], currentPc, st.print(), mem.print(), c.quotaLeft, c.quotaRefund)
			fmt.Println("--------------------")
		}

		if operation.returns {
			vm.returnData = res
		}

		switch {
		case err != nil:
			return nil, err
		case operation.halts:
			return res, nil
		case operation.reverts:
			return res, ErrExecutionReverted
		case !operation.jumps:
			pc++
		}
	}
	return nil, nil
}
