package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/vm/util"
	"sync/atomic"
)

type interpreter struct {
	instructionSet [256]operation
}

var (
	simpleInterpreter         = &interpreter{simpleInstructionSet}
	offchainSimpleInterpreter = &interpreter{offchainSimpleInstructionSet}
	randInterpreter           = &interpreter{randInstructionSet}
	offchainRandInterpreter   = &interpreter{offchainRandInstructionSet}
	earthInterpreter          = &interpreter{earthInstructionSet}
	offchainEarthInterpreter  = &interpreter{offchainEarthInstructionSet}
)

func newInterpreter(blockHeight uint64, offChain bool) *interpreter {
	if fork.IsEarthFork(blockHeight) {
		if offChain {
			return offchainEarthInterpreter
		}
		return earthInterpreter
	}
	if fork.IsSeedFork(blockHeight) {
		if offChain {
			return offchainRandInterpreter
		}
		return randInterpreter
	}
	if offChain {
		return offchainSimpleInterpreter
	}
	return simpleInterpreter
}

func (i *interpreter) runLoop(vm *VM, c *contract) (ret []byte, err error) {
	c.returnData = nil
	var (
		op   opCode
		mem  = newMemory()
		st   = newStack()
		pc   = uint64(0)
		cost uint64
		flag bool
	)

	for atomic.LoadInt32(&vm.abort) == 0 {
		currentPc := pc
		op = c.getOp(pc)
		operation := i.instructionSet[op]

		if !operation.valid {
			nodeConfig.log.Error("invalid opcode", "op", int(op))
			return nil, util.ErrInvalidOpCode
		}

		if err := operation.validateStack(st); err != nil {
			return nil, err
		}

		var memorySize uint64
		if operation.memorySize != nil {
			memSize, overflow := helper.BigUint64(operation.memorySize(st))
			if overflow {
				return nil, util.ErrMemSizeOverflow
			}
			if memorySize, overflow = helper.SafeMul(helper.ToWordSize(memSize), helper.WordSize); overflow {
				return nil, util.ErrMemSizeOverflow
			}
		}

		cost, flag, err = operation.gasCost(vm, c, st, mem, memorySize)
		if err != nil {
			return nil, err
		}
		c.quotaLeft, err = util.UseQuotaWithFlag(c.quotaLeft, cost, flag)
		if err != nil {
			return nil, err
		}

		if memorySize > 0 {
			mem.resize(memorySize)
		}

		res, err := operation.execute(&pc, vm, c, mem, st)

		if nodeConfig.IsDebug {
			currentCode := ""
			if currentPc < uint64(len(c.code)) {
				currentCode = hex.EncodeToString(c.code[currentPc:])
			}
			storageMap, err := c.db.DebugGetStorage()
			if err != nil {
				nodeConfig.interpreterLog.Error("vm step, get storage failed")
			}
			nodeConfig.interpreterLog.Info("vm step",
				"blockType", c.block.BlockType,
				"address", c.block.AccountAddress.String(),
				"height", c.block.Height,
				"fromHash", c.block.FromBlockHash.String(),
				"\ncurrent code", currentCode,
				"\nop", opCodeToString[op],
				"pc", currentPc,
				"quotaLeft", c.quotaLeft,
				"\nstack", st.print(),
				"\nmemory", mem.print(),
				"\nstorage", util.PrintMap(storageMap))
		}

		if operation.returns {
			c.returnData = res
		}

		switch {
		case err != nil:
			return nil, err
		case operation.halts:
			return res, nil
		case operation.reverts:
			return res, util.ErrExecutionReverted
		case !operation.jumps:
			pc++
		}
	}
	panic(util.ErrExecutionCanceled)
}
