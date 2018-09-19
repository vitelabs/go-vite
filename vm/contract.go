package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_context"
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
	block                  *vm_context.VmAccountBlock
	quotaLeft, quotaRefund uint64
	intPool                *intPool
	i                      *interpreter
}

func newContract(caller types.Address, address types.Address, block *vm_context.VmAccountBlock, quotaLeft, quotaRefund uint64) *contract {
	return &contract{caller: caller,
		address:     address,
		block:       block,
		quotaLeft:   quotaLeft,
		quotaRefund: quotaRefund,
		jumpdests:   make(destinations),
		i:           newInterpreter()}
}

func (c *contract) copyContract() *contract {
	return &contract{caller: c.caller,
		address:     c.address,
		block:       c.block,
		quotaLeft:   c.quotaLeft,
		quotaRefund: c.quotaRefund,
		jumpdests:   make(destinations),
		i:           c.i}
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

	return c.i.Run(vm, c)
}
