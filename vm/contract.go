package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

var (
	logger = log15.New("type", "1", "appkey", "govite", "group", "msg", "name", "effectivemsg", "metric", "1", "class", "vm")
)

type contract struct {
	jumpdests       destinations
	data            []byte
	code            []byte
	codeAddr        types.Address
	block           *ledger.AccountBlock
	db              vm_db.VmDb
	sendBlock       *ledger.AccountBlock
	quotaLeft       uint64
	intPool         *util.IntPool
	returnData      []byte
	storageModified map[string]interface{}
}

func newContract(block *ledger.AccountBlock, db vm_db.VmDb, sendBlock *ledger.AccountBlock, data []byte, quotaLeft uint64) *contract {
	return &contract{
		block:           block,
		db:              db,
		sendBlock:       sendBlock,
		data:            data,
		quotaLeft:       quotaLeft,
		jumpdests:       make(destinations),
		storageModified: make(map[string]interface{}),
	}
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
	c.intPool = util.PoolOfIntPools.Get()
	defer func() {
		util.PoolOfIntPools.Put(c.intPool)
		c.intPool = nil
	}()

	return vm.i.runLoop(vm, c)
}
