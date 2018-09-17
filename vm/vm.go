/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"bytes"
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"sync/atomic"
)

type VMConfig struct {
	Debug bool
}

type VM struct {
	VMConfig

	abort          int32
	instructionSet [256]operation
	blockList      []*vm_context.VmAccountBlock
	returnData     []byte
}

func NewVM() *VM {
	return &VM{instructionSet: simpleInstructionSet}
}

func (vm *VM) Run(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	// TODO copy block
	switch block.AccountBlock.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		// block data, amount, tokenId, fee is already changed to send block data by generator
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(block, sendBlock, vm.calcCreateQuota(block.AccountBlock.Fee))
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(block, sendBlock)
		}
	case ledger.BlockTypeSendCreate:
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountBlock.AccountAddress, block)
		block, err = vm.sendCreate(block, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return []*vm_context.VmAccountBlock{block}, NoRetry, nil
		}
	case ledger.BlockTypeSendCall:
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountBlock.AccountAddress, block)
		block, err = vm.sendCall(block, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return []*vm_context.VmAccountBlock{block}, NoRetry, nil
		}
	}

	return nil, NoRetry, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	// check can make transaction
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	contractFee, err := calcContractFee(block.AccountBlock.Data)
	if err != nil {
		return nil, ErrInvalidData
	}
	gid, _ := types.BytesToGid(block.AccountBlock.Data[:types.GidSize])
	if !isExistGid(block.VmContext, gid) {
		return nil, ErrInvalidData
	}
	if !vm.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
		return nil, ErrInsufficientBalance
	}
	// create address
	contractAddr := createContractAddress(block.AccountBlock.AccountAddress, block.AccountBlock.Height, block.AccountBlock.PrevHash, block.AccountBlock.Data, block.AccountBlock.SnapshotHash)

	if block.VmContext.IsAddressExisted(&contractAddr) {
		return nil, ErrContractAddressCreationFail
	}
	block.AccountBlock.Fee = contractFee
	// sub balance and service fee
	block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	if block.AccountBlock.Fee != nil {
		block.VmContext.SubBalance(ledger.ViteTokenId(), block.AccountBlock.Fee)
	}
	vm.updateBlock(block, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	block.AccountBlock.ToAddress = contractAddr
	block.VmContext.SetContractGid(&gid, &contractAddr)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock, quotaTotal uint64) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	quotaLeft := quotaTotal
	if block.VmContext.IsAddressExisted(&block.AccountBlock.AccountAddress) {
		return nil, NoRetry, ErrAddressCollision
	}
	// check can make transaction
	cost, err := intrinsicGasCost(nil, true)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, NoRetry, err
	}

	vm.blockList = []*vm_context.VmAccountBlock{block}

	// create contract account and add balance
	block.VmContext.AddBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)

	blockData := block.AccountBlock.Data
	block.AccountBlock.Data = blockData[types.GidSize:]
	defer func() { block.AccountBlock.Data = blockData }()

	// init contract state and set contract code
	c := newContract(sendBlock.AccountAddress, block.AccountBlock.AccountAddress, block, quotaLeft, 0)
	c.setCallCode(block.AccountBlock.AccountAddress, block.AccountBlock.Data)
	code, err := c.run(vm)
	if err == nil {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = useQuota(c.quotaLeft, codeCost)
		if err == nil {
			codeHash, _ := types.BytesToHash(code)
			block.VmContext.SetContractCode(code)
			vm.updateBlock(block, nil, 0, codeHash.Bytes())
			err = vm.doSendBlockList(quotaTotal - block.AccountBlock.Quota)
			if err == nil {
				return vm.blockList, NoRetry, nil
			}
		}
	}

	vm.revert(block)
	return nil, NoRetry, err
}

func (vm *VM) sendCall(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok := getPrecompiledContract(block.AccountBlock.ToAddress, block.AccountBlock.Data); ok {
		var err error
		block.AccountBlock.Fee = p.getFee(vm, block)
		if !vm.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
			return nil, ErrInsufficientBalance
		}
		quotaLeft, err = p.doSend(vm, block, quotaLeft)
		if err != nil {
			return nil, err
		}
		block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
		block.VmContext.SubBalance(ledger.ViteTokenId(), block.AccountBlock.Fee)
	} else {
		block.AccountBlock.Fee = helper.Big0
		cost, err := intrinsicGasCost(block.AccountBlock.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = useQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
		if !vm.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
			return nil, ErrInsufficientBalance
		}
		block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	}
	var quota uint64
	if isPrecompiledContractAddress(block.AccountBlock.AccountAddress) {
		quota = 0
	} else {
		quota = quotaUsed(quotaTotal, quotaAddition, quotaLeft, 0, nil)
	}
	vm.updateBlock(block, nil, quota, nil)
	return block, nil

}

func (vm *VM) receiveCall(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	if p, ok := getPrecompiledContract(block.AccountBlock.AccountAddress, block.AccountBlock.Data); ok {
		vm.blockList = []*vm_context.VmAccountBlock{block}
		block.VmContext.AddBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
		err := p.doReceive(vm, block, sendBlock)
		if err == nil {
			vm.updateBlock(block, err, 0, nil)
			err = vm.doSendBlockList(txGas)
			if err == nil {
				return vm.blockList, NoRetry, nil
			}
		}
		vm.revert(block)
		vm.updateBlock(block, err, 0, nil)
		return vm.blockList, NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountBlock.AccountAddress, block)
		quotaLeft := quotaTotal
		quotaRefund := uint64(0)
		cost, err := intrinsicGasCost(nil, false)
		if err != nil {
			return nil, NoRetry, err
		}
		quotaLeft, err = useQuota(quotaLeft, cost)
		if err != nil {
			return nil, Retry, err
		}
		vm.blockList = []*vm_context.VmAccountBlock{block}
		// add balance, create account if not exist
		block.VmContext.AddBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
		// do transfer transaction if account code size is zero
		code := block.VmContext.GetContractCode(&block.AccountBlock.AccountAddress)
		if len(code) == 0 {
			vm.updateBlock(block, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
			return vm.blockList, NoRetry, nil
		}
		// run code
		c := newContract(sendBlock.AccountAddress, block.AccountBlock.AccountAddress, block, quotaLeft, quotaRefund)
		c.setCallCode(block.AccountBlock.AccountAddress, code)
		_, err = c.run(vm)
		if err == nil {
			vm.updateBlock(block, nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), nil)
			err = vm.doSendBlockList(quotaTotal - quotaAddition - block.AccountBlock.Quota)
			if err == nil {
				return vm.blockList, NoRetry, nil
			}
		}

		vm.revert(block)
		vm.updateBlock(block, err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		return vm.blockList, err == ErrOutOfQuota, err
	}
}

func (vm *VM) sendReward(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	// check can make transaction
	quotaLeft := quotaTotal
	cost, err := intrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), contracts.AddressRegister.Bytes()) && !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), contracts.AddressMintage.Bytes()) {
		return nil, ErrInvalidData
	}
	vm.updateBlock(block, nil, 0, nil)
	return block, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	cNew := newContract(c.caller, c.address, c.block, c.quotaLeft, c.quotaRefund)
	cNew.setCallCode(contractAddr, c.block.VmContext.GetContractCode(&contractAddr))
	ret, err = cNew.run(vm)
	c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
	return ret, err
}

func (vm *VM) calcCreateQuota(fee *big.Int) uint64 {
	quota := new(big.Int).Div(fee, quotaByCreateFeeAttov)
	if quota.IsUint64() {
		return helper.Min(quotaLimitForTransaction, quota.Uint64())
	}
	return quotaLimitForTransaction
}

func (vm *VM) quotaLeft(addr types.Address, block *vm_context.VmAccountBlock) (uint64, uint64) {
	// quotaInit = pledge amount of account address at current snapshot block status(attov) / quotaByPledge
	// get extra quota if calc PoW before a send transaction
	quotaInit := helper.Min(new(big.Int).Div(contracts.GetPledgeAmount(block.VmContext, addr), quotaByPledge).Uint64(), quotaLimit)
	quotaAddition := uint64(0)
	if len(block.AccountBlock.Nonce) > 0 {
		quotaAddition = quotaForPoW
	}
	prevHash := block.AccountBlock.PrevHash
	for {
		prevBlock := block.VmContext.GetAccountBlockByHash(&prevHash)
		if prevBlock != nil && bytes.Equal(block.AccountBlock.SnapshotHash.Bytes(), prevBlock.SnapshotHash.Bytes()) {
			// quick fail on a receive error block referencing to the same snapshot block
			// only one block gets extra quota when referencing to the same snapshot block
			if prevBlock.BlockType == ledger.BlockTypeReceiveError || (len(prevBlock.Nonce) > 0 && len(block.AccountBlock.Nonce) > 0) {
				return 0, 0
			}
			quotaInit = quotaInit - prevBlock.Quota
			prevHash = prevBlock.PrevHash
		} else {
			if quotaLimit-quotaAddition < quotaInit {
				quotaAddition = quotaLimit - quotaInit
				quotaInit = quotaLimit
			} else {
				quotaInit = quotaInit + quotaAddition
			}
			return quotaInit, quotaAddition
		}
	}
}

func (vm *VM) updateBlock(block *vm_context.VmAccountBlock, err error, quota uint64, result []byte) {
	block.AccountBlock.Quota = quota
	block.AccountBlock.StateHash = *block.VmContext.GetStorageHash()
	if block.AccountBlock.BlockType == ledger.BlockTypeReceive || block.AccountBlock.BlockType == ledger.BlockTypeReceiveError {
		// data = fixed byte of execution result + result
		if err == nil {
			block.AccountBlock.Data = append(DataResultPrefixSuccess, result...)
		} else if err == ErrExecutionReverted {
			block.AccountBlock.Data = append(DataResultPrefixRevert, result...)
		} else {
			block.AccountBlock.Data = append(DataResultPrefixFail, result...)
		}

		block.AccountBlock.LogHash = block.VmContext.GetLogListHash()
		if err == ErrOutOfQuota {
			block.AccountBlock.BlockType = ledger.BlockTypeReceiveError
		} else {
			block.AccountBlock.BlockType = ledger.BlockTypeReceive
		}
	}
}

func (vm *VM) doSendBlockList(quotaLeft uint64) (err error) {
	db := vm.blockList[0].VmContext
	for i, block := range vm.blockList[1:] {
		db = db.CopyAndFreeze()
		block.VmContext = db
		switch block.AccountBlock.BlockType {
		case ledger.BlockTypeSendCall:
			vm.blockList[i+1], err = vm.sendCall(block, quotaLeft, 0)
			if err != nil {
				return err
			}
		case ledger.BlockTypeSendReward:
			vm.blockList[i+1], err = vm.sendReward(block, quotaLeft, 0)
			if err != nil {
				return err
			}
		}
		quotaLeft = quotaLeft - vm.blockList[i+1].AccountBlock.Quota
	}
	return nil
}

func (vm *VM) revert(block *vm_context.VmAccountBlock) {
	vm.blockList = vm.blockList[:1]
	vm.returnData = nil
	block.VmContext.Reset()
}

func (vm *VM) canTransfer(db vmctxt_interface.VmDatabase, addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
	if feeAmount == nil || feeAmount.Sign() == 0 {
		return tokenAmount.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0
	} else if IsViteToken(tokenTypeId) {
		balance := new(big.Int).Add(tokenAmount, feeAmount)
		return balance.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0
	} else {
		return tokenAmount.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0 && feeAmount.Cmp(db.GetBalance(&addr, ledger.ViteTokenId())) <= 0
	}
}

func calcContractFee(data []byte) (*big.Int, error) {
	return contractFee, nil
}

func quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund uint64, err error) uint64 {
	if err == ErrOutOfQuota {
		return quotaTotal - quotaAddition
	} else if err != nil {
		if quotaTotal-quotaLeft < quotaAddition {
			return 0
		} else {
			return quotaTotal - quotaAddition - quotaLeft
		}
	} else {
		if quotaTotal-quotaLeft < quotaAddition {
			return 0
		} else {
			return quotaTotal - quotaLeft - quotaAddition - helper.Min(quotaRefund, (quotaTotal-quotaAddition-quotaLeft)/2)
		}
	}
}

func createContractAddress(addr types.Address, height uint64, prevHash types.Hash, code []byte, snapshotHash types.Hash) types.Address {
	return types.CreateContractAddress(addr.Bytes(), new(big.Int).SetUint64(height).Bytes(), prevHash.Bytes(), code, snapshotHash.Bytes())
}

func isExistGid(db vmctxt_interface.VmDatabase, gid types.Gid) bool {
	value := db.GetStorage(&contracts.AddressConsensusGroup, contracts.GetConsensusGroupKey(gid))
	return len(value) > 0
}

func makeSendBlock(block *ledger.AccountBlock, toAddress types.Address, blockType byte, amount *big.Int, tokenId types.TokenTypeId, data []byte) *ledger.AccountBlock {
	return &ledger.AccountBlock{AccountAddress: block.AccountAddress, ToAddress: toAddress, BlockType: blockType, Amount: amount, TokenId: tokenId, Height: block.Height + 1, SnapshotHash: block.SnapshotHash, Data: data}
}
