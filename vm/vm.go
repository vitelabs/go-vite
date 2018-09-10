package vm

import "sync/atomic"

type VMConfig struct {
	Debug bool
}

type VM struct {
	VMConfig
	Db          VmDatabase
	abort       int32
	createBlock CreateAccountBlockFunc
}

func NewVM(db VmDatabase, createBlockFunc CreateAccountBlockFunc) *VM {
	return &VM{Db: db, createBlock: createBlockFunc}
}

func (vm *VM) Run(block VmAccountBlock) (blockList []VmAccountBlock, isRetry bool, err error) {
	return nil, false, nil
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}
