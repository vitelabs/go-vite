/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"bytes"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"sync/atomic"
)

type VMConfig struct {
	Debug bool
}

type Log struct {
	Address       types.Address
	Topics        []types.Hash
	Data          []byte
	AccountHeight uint64
}

type SendTransaction struct {
	From        types.Address
	To          types.Address
	Data        []byte
	Amount      *big.Int
	TokenTypeId types.TokenTypeId
	Depth       int
	Height      *big.Int
}

type VM struct {
	VMConfig
	StateDb     Database
	createBlock CreateBlockFunc

	abort          int32
	intPool        *intPool
	instructionSet [256]operation
	logList        []*Log
	blockList      []VmBlock
	returnData     []byte
}

func Run(stateDb Database, createBlockFunc CreateBlockFunc, config VMConfig, block VmBlock) (blockList []VmBlock, logList []*Log, err error) {
	vm := &VM{StateDb: stateDb, createBlock: createBlockFunc, instructionSet: simpleInstructionSet, logList: make([]*Log, 0), VMConfig: config}
	switch block.TxType() {
	case TxTypeReceive, TxTypeReceiveError:
		sendBlock := vm.StateDb.AccountBlock(block.From(), block.FromHash())
		block.SetData(sendBlock.Data())
		// TODO sendBlock not exist
		if sendBlock.TxType() == TxTypeSendCreate {
			return vm.receiveCreate(block)
		} else if sendBlock.TxType() == TxTypeSendCall {
			return vm.receiveCall(block)
		}
	case TxTypeSendCreate:
		block, err = vm.sendCreate(block)
		if err != nil {
			return []VmBlock{}, nil, err
		} else {
			return []VmBlock{block}, nil, nil
		}
	case TxTypeSendCall:
		block, err = vm.sendCall(block)
		if err != nil {
			return []VmBlock{}, nil, err
		} else {
			return []VmBlock{block}, nil, nil
		}
	}
	return nil, nil, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block VmBlock) (VmBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.From(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data(), true)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	createFee := calcCreateContractFee(block.Data())
	if !canTransfer(vm.StateDb, block.From(), block.TokenId(), block.Amount(), createFee) {
		return nil, ErrInsufficientBalance
	}
	// create address
	contractAddr, err := createAddress(block.From(), block.Height(), block.Data())

	if err != nil || vm.StateDb.IsExistAddress(contractAddr) {
		return nil, ErrContractAddressCreationFail
	}
	// sub balance and service fee
	vm.StateDb.SubBalance(block.From(), block.TokenId(), block.Amount())
	vm.StateDb.SubBalance(block.From(), viteTokenTypeId, createFee)
	block.SetCreateFee(createFee)
	vm.updateBlock(block, block.From(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	block.SetTo(contractAddr)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block VmBlock) (blockList []VmBlock, logList []*Log, err error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.To(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data(), true)
	if err != nil {
		// do not return block when receive account don't have enough quota to start a transaction
		return []VmBlock{}, nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		// do not return block when receive account don't have enough quota to start a transaction
		return []VmBlock{}, nil, err
	}

	if vm.StateDb.IsExistAddress(block.To()) {
		return []VmBlock{}, nil, ErrAddressCollision
	}

	vm.blockList = []VmBlock{block}

	// create contract account and add balance
	vm.StateDb.CreateAccount(block.To())
	vm.StateDb.AddBalance(block.To(), block.TokenId(), block.Amount())

	if block.Depth() > callCreateDepth {
		vm.StateDb.DeleteAccount(block.To())
		vm.updateBlock(block, block.To(), ErrDepth, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, ErrDepth), nil)
		return vm.blockList, nil, ErrDepth
	}

	// init contract state and set contract code
	c := newContract(block.From(), block.To(), block, quotaLeft, quotaRefund)
	c.setCallCode(block.To(), types.DataHash(block.Data()), block.Data())
	code, err := c.run(vm)
	if err == nil {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = useQuota(c.quotaLeft, codeCost)
		if err == nil {
			codeHash, _ := types.BytesToHash(code)
			vm.StateDb.SetContractCode(block.To(), code, codeHash)
			vm.updateBlock(block, block.To(), nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), codeHash.Bytes())
			err = vm.doSendBlockList()
			if err == nil {
				return vm.blockList, vm.logList, nil
			}
		}
	}

	// revert if out of quota, retry later; refund and delete account otherwise.
	vm.revert()
	if err == ErrOutOfQuota {
		vm.updateBlock(block, block.To(), err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		return []VmBlock{block}, nil, err
	} else {
		vm.StateDb.CreateAccount(block.To())
		vm.StateDb.AddBalance(block.To(), block.TokenId(), block.Amount())

		if block.Amount().Cmp(big0) > 0 {
			refundBlock := vm.createBlock(block.To(), block.From(), TxTypeSendCall, block.Depth()+1)
			refundBlock.SetTokenId(block.TokenId())
			refundBlock.SetAmount(block.Amount())
			vm.blockList = append(vm.blockList, refundBlock)
		}
		vm.StateDb.DeleteAccount(block.To())
		vm.updateBlock(block, block.To(), err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		sendErr := vm.doSendBlockList()
		if sendErr == nil {
			return vm.blockList, nil, err
		} else {
			vm.blockList = vm.blockList[:1]
			vm.updateBlock(block, block.To(), sendErr, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, sendErr), nil)
			return vm.blockList, nil, sendErr
		}
	}
}

func (vm *VM) sendCall(block VmBlock) (VmBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.From(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data(), false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if !canTransfer(vm.StateDb, block.From(), block.TokenId(), block.Amount(), big0) {
		return nil, ErrInsufficientBalance
	}
	// sub balance
	vm.StateDb.SubBalance(block.From(), block.TokenId(), block.Amount())
	vm.updateBlock(block, block.From(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return block, nil
}

func (vm *VM) receiveCall(block VmBlock) (blockList []VmBlock, logList []*Log, err error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.To(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data(), false)
	if err != nil {
		// do not return block when receive account cannot start a transaction
		return []VmBlock{}, nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		// do not return block when receive account don't have enough quota to start a transaction
		return []VmBlock{}, nil, err
	}
	vm.blockList = []VmBlock{block}
	// create genesis block when accepting first receive transaction
	if !vm.StateDb.IsExistAddress(block.To()) {
		vm.StateDb.CreateAccount(block.To())
	}
	vm.StateDb.AddBalance(block.To(), block.TokenId(), block.Amount())
	// do transfer transaction if account code size is zero
	if vm.StateDb.ContractCodeSize(block.To()) == 0 {
		vm.updateBlock(block, block.To(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
		return []VmBlock{block}, nil, nil
	}
	// do nothing but add balance if reaches the maximum call or create depth
	if block.Depth() > callCreateDepth {
		vm.updateBlock(block, block.To(), ErrDepth, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, ErrDepth), nil)
		return []VmBlock{block}, nil, ErrDepth
	}
	// run code
	c := newContract(block.From(), block.To(), block, quotaLeft, quotaRefund)
	c.setCallCode(block.To(), vm.StateDb.ContractCodeHash(block.To()), vm.StateDb.ContractCode(block.To()))
	_, err = c.run(vm)
	if err == nil {
		vm.updateBlock(block, block.To(), nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), nil)
		err = vm.doSendBlockList()
		if err == nil {
			return vm.blockList, vm.logList, nil
		}
	}

	vm.revert()
	// revert if out of quota, retry later; refund and delete account otherwise.
	if err == ErrOutOfQuota {
		vm.updateBlock(block, block.To(), err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		return vm.blockList, nil, ErrOutOfQuota
	} else {
		// redo add balance
		if !vm.StateDb.IsExistAddress(block.To()) {
			vm.StateDb.CreateAccount(block.To())
		}
		vm.StateDb.AddBalance(block.To(), block.TokenId(), block.Amount())

		if block.Amount().Cmp(big0) > 0 {
			refundBlock := vm.createBlock(block.To(), block.From(), TxTypeSendCall, block.Depth()+1)
			refundBlock.SetTokenId(block.TokenId())
			refundBlock.SetAmount(block.Amount())
			vm.blockList = append(vm.blockList, refundBlock)
		}
		vm.updateBlock(block, block.To(), err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		sendErr := vm.doSendBlockList()
		if sendErr == nil {
			return vm.blockList, nil, err
		} else {
			vm.blockList = vm.blockList[:1]
			vm.updateBlock(block, block.To(), sendErr, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, sendErr), nil)
			return vm.blockList, nil, sendErr
		}
	}
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	cNew := newContract(c.caller, c.address, c.block, c.quotaLeft, c.quotaRefund)
	cNew.setCallCode(contractAddr, vm.StateDb.ContractCodeHash(contractAddr), vm.StateDb.ContractCode(contractAddr))
	ret, err = cNew.run(vm)
	c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
	return ret, err
}

func (vm *VM) quotaLeft(addr types.Address, block VmBlock) (quotaInit, quotaAddition uint64) {
	// TODO calculate quota, use max for test
	// TODO calculate quota addition
	quotaInit = maxUint64
	quotaAddition = 0
	for _, block := range vm.blockList {
		if quotaInit <= block.Quota() {
			return 0, 0
		} else {
			quotaInit = quotaInit - block.Quota()
		}
	}
	prevHash := block.PrevHash()
	if len(vm.blockList) > 0 {
		prevHash = vm.blockList[0].PrevHash()
	}
	for {
		prevBlock := vm.StateDb.AccountBlock(addr, prevHash)
		if prevBlock != nil && bytes.Equal(block.SnapshotHash().Bytes(), prevBlock.SnapshotHash().Bytes()) {
			quotaInit = quotaInit - prevBlock.Quota()
		} else {
			if maxUint64-quotaAddition > quotaInit {
				return maxUint64, maxUint64 - quotaInit
			} else {
				return quotaInit + quotaAddition, quotaAddition
			}
		}
	}
}

func quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund uint64, err error) uint64 {
	if err == ErrOutOfQuota {
		return quotaTotal - quotaAddition
	} else if err != nil {
		return quotaTotal - quotaAddition - quotaLeft
	} else {
		return quotaTotal - quotaAddition - quotaLeft - min(quotaRefund, (quotaTotal-quotaAddition-quotaLeft)/2)
	}
}

func (vm *VM) updateBlock(block VmBlock, addr types.Address, err error, quota uint64, result []byte) {
	block.SetQuota(quota)
	// TODO data = fixed byte of err + result
	block.SetData(result)
	block.SetStateHash(vm.StateDb.StateHash(addr))
	if block.TxType() == TxTypeReceive || block.TxType() == TxTypeReceiveError {
		if err == ErrOutOfQuota || err == ErrContractAddressCreationFail {
			block.SetTxType(TxTypeReceiveError)
		} else {
			block.SetTxType(TxTypeReceive)
		}
		if len(vm.blockList) > 1 {
			for _, sendBlock := range vm.blockList[1:] {
				block.AppendSummaryHash(sendBlock.SummaryHash())
			}
		}
	}
}

func (vm *VM) doSendBlockList() (err error) {
	for i, block := range vm.blockList[1:] {
		if block.To != nil {
			vm.blockList[i], err = vm.sendCall(block)
			if err != nil {
				return err
			}
		} else {
			vm.blockList[i], err = vm.sendCreate(block)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *VM) revert() {
	vm.blockList = vm.blockList[:1]
	vm.logList = nil
	vm.returnData = nil
	vm.StateDb.Revert()
}

func calcCreateContractFee(InputData []byte) *big.Int {
	// TODO calculate service fee for create contract, use 0 for test
	return big0
}

// TODO set vite token type id
var viteTokenTypeId = types.TokenTypeId{}

func canTransfer(db Database, addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
	return tokenAmount.Cmp(db.Balance(addr, tokenTypeId)) <= 0 && feeAmount.Cmp(db.Balance(addr, viteTokenTypeId)) <= 0
}

func createAddress(addr types.Address, height *big.Int, code []byte) (types.Address, error) {
	var a types.Address
	dataBytes := append(addr.Bytes(), height.Bytes()...)
	dataBytes = append(dataBytes, code...)
	addressHash := types.DataHash(dataBytes)
	err := a.SetBytes(addressHash[12:])
	return a, err
}
