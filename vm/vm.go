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
	// list of topics provided by the contract
	Topics []types.Hash
	// supplied by the contract, usually ABI-encoded
	Data []byte
}

type VM struct {
	VMConfig
	StateDb     Database
	createBlock CreateAccountBlockFunc

	abort          int32
	intPool        *intPool
	instructionSet [256]operation
	logList        []*Log
	blockList      []VmAccountBlock
	returnData     []byte
}

func NewVM(stateDb Database, createBlockFunc CreateAccountBlockFunc, config VMConfig) *VM {
	return &VM{StateDb: stateDb, createBlock: createBlockFunc, instructionSet: simpleInstructionSet, logList: make([]*Log, 0), VMConfig: config}
}

func (vm *VM) Run(block VmAccountBlock) (blockList []VmAccountBlock, logList []*Log, isRetry bool, err error) {
	switch block.BlockType() {
	case BlockTypeReceive, BlockTypeReceiveError:
		sendBlock := vm.StateDb.AccountBlock(block.AccountAddress(), block.FromBlockHash())
		block.SetData(sendBlock.Data())
		if sendBlock.BlockType() == BlockTypeSendCreate {
			return vm.receiveCreate(block, vm.calcCreateQuota(sendBlock.CreateFee()))
		} else if sendBlock.BlockType() == BlockTypeSendCall {
			return vm.receiveCall(block)
		} else if sendBlock.BlockType() == BlockTypeSendMintage {
			// TODO
		}
	case BlockTypeSendCreate:
		block, err = vm.sendCreate(block)
		if err != nil {
			return nil, nil, noRetry, err
		} else {
			return []VmAccountBlock{block}, nil, noRetry, nil
		}
	case BlockTypeSendCall:
		block, err = vm.sendCall(block)
		if err != nil {
			return nil, nil, noRetry, err
		} else {
			return []VmAccountBlock{block}, nil, noRetry, nil
		}
	case BlockTypeSendMintage:
		// TODO
	}
	return nil, nil, noRetry, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block VmAccountBlock) (VmAccountBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress(), block)
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
	if !checkContractFee(block.CreateFee()) {
		return nil, ErrInvalidContractFee
	}
	if !vm.canTransfer(block.AccountAddress(), block.TokenId(), block.Amount(), block.CreateFee()) {
		return nil, ErrInsufficientBalance
	}
	// create address
	contractAddr, err := createAddress(block.AccountAddress(), block.Height(), block.Data(), block.SnapshotHash())

	if err != nil || vm.StateDb.IsExistAddress(contractAddr) {
		return nil, ErrContractAddressCreationFail
	}
	// sub balance and service fee
	vm.StateDb.SubBalance(block.AccountAddress(), block.TokenId(), block.Amount())
	vm.StateDb.SubBalance(block.AccountAddress(), viteTokenTypeId, block.CreateFee())
	vm.updateBlock(block, block.AccountAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	block.SetToAddress(contractAddr)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block VmAccountBlock, quotaLeft uint64) (blockList []VmAccountBlock, logList []*Log, isRetry bool, err error) {
	if vm.StateDb.IsExistAddress(block.ToAddress()) {
		return nil, nil, noRetry, ErrAddressCollision
	}
	// check can make transaction
	cost, err := intrinsicGasCost(nil, true)
	if err != nil {
		return nil, nil, noRetry, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, nil, noRetry, err
	}

	vm.blockList = []VmAccountBlock{block}

	if block.Depth() > callCreateDepth {
		vm.updateBlock(block, block.ToAddress(), ErrDepth, 0, nil)
		return vm.blockList, nil, noRetry, ErrDepth
	}

	// create contract account and add balance
	vm.StateDb.CreateAccount(block.ToAddress())
	vm.StateDb.AddBalance(block.ToAddress(), block.TokenId(), block.Amount())

	// init contract state and set contract code
	c := newContract(block.AccountAddress(), block.ToAddress(), block, quotaLeft, 0)
	c.setCallCode(block.ToAddress(), block.Data())
	code, err := c.run(vm)
	if err == nil {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = useQuota(c.quotaLeft, codeCost)
		if err == nil {
			codeHash, _ := types.BytesToHash(code)
			vm.StateDb.SetContractCode(block.ToAddress(), code)
			vm.updateBlock(block, block.ToAddress(), nil, 0, codeHash.Bytes())
			err = vm.doSendBlockList()
			if err == nil {
				return vm.blockList, vm.logList, noRetry, nil
			}
		}
	}

	vm.revert()
	vm.StateDb.CreateAccount(block.ToAddress())
	vm.updateBlock(block, block.ToAddress(), err, 0, nil)
	return vm.blockList, nil, noRetry, err
}

func (vm *VM) sendCall(block VmAccountBlock) (VmAccountBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress(), block)
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
	if !vm.canTransfer(block.AccountAddress(), block.TokenId(), block.Amount(), big0) {
		return nil, ErrInsufficientBalance
	}
	// sub balance
	vm.StateDb.SubBalance(block.AccountAddress(), block.TokenId(), block.Amount())
	vm.updateBlock(block, block.AccountAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return block, nil
}

func (vm *VM) receiveCall(block VmAccountBlock) (blockList []VmAccountBlock, logList []*Log, isRetry bool, err error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.ToAddress(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(nil, false)
	if err != nil {
		return nil, nil, noRetry, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, nil, retry, err
	}
	vm.blockList = []VmAccountBlock{block}
	// create genesis block when accepting first receive transaction
	if !vm.StateDb.IsExistAddress(block.ToAddress()) {
		vm.StateDb.CreateAccount(block.ToAddress())
	}
	if block.Depth() > callCreateDepth {
		vm.updateBlock(block, block.ToAddress(), ErrDepth, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, ErrDepth), nil)
		return vm.blockList, nil, noRetry, ErrDepth
	}
	vm.StateDb.AddBalance(block.ToAddress(), block.TokenId(), block.Amount())
	// do transfer transaction if account code size is zero
	code := vm.StateDb.ContractCode(block.ToAddress())
	if len(code) == 0 {
		vm.updateBlock(block, block.ToAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
		return vm.blockList, nil, noRetry, nil
	}
	// run code
	c := newContract(block.AccountAddress(), block.ToAddress(), block, quotaLeft, quotaRefund)
	c.setCallCode(block.ToAddress(), code)
	_, err = c.run(vm)
	if err == nil {
		vm.updateBlock(block, block.ToAddress(), nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), nil)
		err = vm.doSendBlockList()
		if err == nil {
			return vm.blockList, vm.logList, noRetry, nil
		}
	}

	vm.revert()
	if !vm.StateDb.IsExistAddress(block.ToAddress()) {
		vm.StateDb.CreateAccount(block.ToAddress())
	}
	vm.updateBlock(block, block.ToAddress(), err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
	return vm.blockList, nil, err == ErrOutOfQuota, err
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	cNew := newContract(c.caller, c.address, c.block, c.quotaLeft, c.quotaRefund)
	cNew.setCallCode(contractAddr, vm.StateDb.ContractCode(contractAddr))
	ret, err = cNew.run(vm)
	c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
	return ret, err
}

func (vm *VM) calcCreateQuota(fee *big.Int) uint64 {
	// TODO calculate quota for create contract receive transaction
	return quotaLimit
}

func (vm *VM) quotaLeft(addr types.Address, block VmAccountBlock) (quotaInit, quotaAddition uint64) {
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
				quotaAddition = maxUint64 - quotaInit
				quotaInit = maxUint64
			} else {
				quotaInit = quotaInit + quotaAddition
			}
			return min(quotaInit, quotaLimit), min(quotaAddition, quotaLimit)
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

func (vm *VM) updateBlock(block VmAccountBlock, addr types.Address, err error, quota uint64, result []byte) {
	block.SetQuota(quota)
	// TODO data = fixed byte of err + result
	block.SetData(result)
	block.SetStateHash(vm.StateDb.StorageHash(addr))
	if block.BlockType() == BlockTypeReceive || block.BlockType() == BlockTypeReceiveError {
		if err == ErrOutOfQuota {
			block.SetBlockType(BlockTypeReceiveError)
		} else {
			block.SetBlockType(BlockTypeReceive)
		}
		if len(vm.blockList) > 1 {
			for _, sendBlock := range vm.blockList[1:] {
				block.AppendSendBlockHash(sendBlock.SummaryHash())
			}
		}
	}
}

func (vm *VM) doSendBlockList() (err error) {
	for i, block := range vm.blockList[1:] {
		if block.ToAddress() != emptyAddress {
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
	vm.StateDb.Rollback()
}

func checkContractFee(fee *big.Int) bool {
	return ContractFeeMin.Cmp(fee) <= 0 && ContractFeeMax.Cmp(fee) >= 0
}

// TODO set vite token type id
var viteTokenTypeId = types.TokenTypeId{}

func (vm *VM) canTransfer(addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
	if bytes.Equal(tokenTypeId.Bytes(), viteTokenTypeId.Bytes()) {
		balance := new(big.Int).Add(tokenAmount, feeAmount)
		return balance.Cmp(vm.StateDb.Balance(addr, tokenTypeId)) <= 0
	} else {
		return tokenAmount.Cmp(vm.StateDb.Balance(addr, tokenTypeId)) <= 0 && feeAmount.Cmp(vm.StateDb.Balance(addr, viteTokenTypeId)) <= 0
	}
}

func createAddress(addr types.Address, height *big.Int, code []byte, snapshotHash types.Hash) (types.Address, error) {
	var a types.Address
	dataBytes := append(addr.Bytes(), height.Bytes()...)
	dataBytes = append(dataBytes, code...)
	dataBytes = append(dataBytes, snapshotHash.Bytes()...)
	addressHash := types.DataHash(dataBytes)
	err := a.SetBytes(addressHash[12:])
	return a, err
}
