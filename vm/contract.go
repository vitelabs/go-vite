package vm

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm/util"
	"sync/atomic"
)

var (
	logger = log15.New("type", "1", "appkey", "govite", "group", "msg", "name", "effectivemsg", "metric", "1", "class", "vm")
)

type contract struct {
	caller                 types.Address
	address                types.Address
	jumpdests              destinations
	code                   []byte
	codeAddr               types.Address
	block                  *ledger.AccountBlock
	quotaLeft, quotaRefund uint64
	intPool                *intPool
}

func newContract(caller types.Address, address types.Address, block *ledger.AccountBlock, quotaLeft, quotaRefund uint64) *contract {
	return &contract{caller: caller,
		address:     address,
		block:       block,
		quotaLeft:   quotaLeft,
		quotaRefund: quotaRefund,
		jumpdests:   make(destinations)}
}

func (c *contract) getOp(n uint64) opCode {
	return opCode(c.getByte(n))
}

func (c *contract) getByte(n uint64) byte {
	if n < uint64(len(c.code)) {
		return c.code[n]
	}

	return 0
}

func (c *contract) setCallCode(addr types.Address, code []byte) {
	c.code = code
	c.codeAddr = addr
}

func (c *contract) run(vm *VM) (ret []byte, err error) {
	if len(c.code) == 0 {
		return nil, nil
	}

	c.intPool = poolOfIntPools.get()
	defer func() {
		poolOfIntPools.put(c.intPool)
		c.intPool = nil
	}()

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
		operation := vm.instructionSet[op]

		if !operation.valid {
			return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
		}

		if err := operation.validateStack(st); err != nil {
			return nil, err
		}

		var memorySize uint64
		if operation.memorySize != nil {
			memSize, overflow := util.BigUint64(operation.memorySize(st))
			if overflow {
				return nil, errGasUintOverflow
			}
			if memorySize, overflow = util.SafeMul(util.ToWordSize(memSize), util.WordSize); overflow {
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
			fmt.Printf("op: %v, pc: %v\nstack: [%v]\nmemo ry: [%v]\nquotaLeft: %v, quotaRefund: %v\n", opCodeToString[op], currentPc, st.print(), mem.print(), c.quotaLeft, c.quotaRefund)
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
