package generator

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_db"
	"runtime/debug"
)

// fixme？？？
func RecoverVmContext(chain vm_db.Chain, block *ledger.AccountBlock, snapshotHash *types.Hash) (vmDbList vm_db.VmDb, resultErr error) {
	var tLog = log15.New("method", "RecoverVmContext")
	vmDb, err := vm_db.NewVmDb(chain, &block.AccountAddress, snapshotHash, &block.PrevHash)
	if err != nil {
		return nil, err
	}
	var sendBlock *ledger.AccountBlock = nil
	if block.IsReceiveBlock() {
		if sendBlock, err = chain.GetAccountBlockByHash(&block.FromBlockHash); sendBlock == nil {
			if err != nil {
				return nil, err
			}
			return nil, ErrGetVmContextValueFailed
		}
	}
	defer func() {
		if err := recover(); err != nil {
			// print stack
			debug.PrintStack()
			errDetail := fmt.Sprintf("block(addr:%v prevHash:%v sbHash:%v )", block.AccountAddress, block.PrevHash, snapshotHash)
			if sendBlock != nil {
				errDetail += fmt.Sprintf("sendBlock(addr:%v hash:%v)", block.AccountAddress, block.Hash)
			}
			tLog.Error(fmt.Sprintf("generator_vm panic error %v", err), "detail", errDetail)
			resultErr = errors.New("generator_vm panic error")
		}
	}()

	newVm := NewVM()
	vmBlock, isRetry, err := newVm.Run(vmDb, block, sendBlock, nil)

	tLog.Debug("vm result", fmt.Sprintf("vmBlock.Hash %v, isRetry %v, err %v", vmBlock.VmDb, isRetry, err))

	return vmBlock.VmDb, err
}
