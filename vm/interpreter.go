package vm

import (
	"sync/atomic"

	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/vm/util"
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
	vep19Interpreter          = &interpreter{vep19InstructionsSet}
	offchainVep19Interpreter  = &interpreter{offchainVep19InstructionSet}
)

func newInterpreter(blockHeight uint64, offChain bool) *interpreter {
	// introduce VEP19 in Version11 upgrade
	if upgrade.IsVersion11Upgrade(blockHeight) {
		if offChain {
			return offchainVep19Interpreter
		}
		nodeConfig.log.Debug("New interpreter on Version11 Fork", "height", blockHeight)
		return vep19Interpreter
	}
	if upgrade.IsEarthUpgrade(blockHeight) {
		if offChain {
			return offchainEarthInterpreter
		}
		nodeConfig.log.Debug("New interpreter on Earth Fork", "height", blockHeight)
		return earthInterpreter
	}
	if upgrade.IsSeedUpgrade(blockHeight) {
		if offChain {
			return offchainRandInterpreter
		}
		nodeConfig.log.Debug("New interpreter on Seed Fork", "height", blockHeight)
		return randInterpreter
	}
	if offChain {
		return offchainSimpleInterpreter
	}
	nodeConfig.log.Debug("New interpreter on init Fork", "height", blockHeight)
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
		nodeConfig.log.Debug("compare code and cost", "operation", operation, "quota left", c.quotaLeft, "cost", cost, "memorySize", memorySize, "err", err)
		if err != nil {
			return nil, err
		}

		if memorySize > 0 {
			mem.resize(memorySize)
		}

		res, err := operation.execute(&pc, vm, c, mem, st)

		if nodeConfig.IsDebug {
			currentCode :=  "[" + opCodeToString[c.getOp(currentPc)] + "]"
			if currentPc > 0 {
				currentCode = opCodeToString[c.getOp(currentPc - 1)] + ", " + currentCode
			}
			if currentPc < uint64(len(c.code) - 1) {
				currentCode = currentCode + ", " + opCodeToString[c.getOp(currentPc + 1)]
			}
			storageMap, err := c.db.DebugGetStorage()
			if err != nil {
				nodeConfig.interpreterLog.Error("vm step, get storage failed")
			}
			nodeConfig.interpreterLog.Info("vm step",
				"blockType", c.block.BlockType,
				"height", c.block.Height,
				"address", c.block.AccountAddress.String(),
				"fromHash", c.block.FromBlockHash.String(),
				"\npc", currentPc,
				"quotaLeft", c.quotaLeft,
				"\ncode", currentCode,
				"\nop", opCodeToString[op],
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
