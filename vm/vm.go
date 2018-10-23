/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"sync/atomic"
	"time"
)

type VMConfig struct {
	Debug bool
}

type NodeConfig struct {
	IsTest    bool
	calcQuota func(db vmctxt_interface.VmDatabase, addr types.Address, pow bool) (quotaTotal uint64, quotaAddition uint64)
	params    VmParams
}

var nodeConfig NodeConfig

func InitVmConfig(isTest bool, isTestParam bool) {
	if isTest {
		nodeConfig = NodeConfig{
			IsTest: isTest,
			calcQuota: func(db vmctxt_interface.VmDatabase, addr types.Address, pow bool) (quotaTotal uint64, quotaAddition uint64) {
				return 1000000, 0
			},
		}
	} else {
		nodeConfig = NodeConfig{
			IsTest: isTest,
			calcQuota: func(db vmctxt_interface.VmDatabase, addr types.Address, pow bool) (quotaTotal uint64, quotaAddition uint64) {
				return quota.CalcQuota(db, addr, pow)
			},
		}
	}
	if isTestParam {
		nodeConfig.params = VmParamsTest
	} else {
		nodeConfig.params = VmParamsMainNet
	}
}

type VM struct {
	VMConfig

	abort     int32
	blockList []*vm_context.VmAccountBlock

	i *Interpreter
}

func NewVM() *VM {
	return &VM{i: simpleInterpreter}
}

func (vm *VM) Run(database vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	defer monitor.LogTime("vm", "run", time.Now())
	blockContext := &vm_context.VmAccountBlock{block.Copy(), database}
	switch block.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		// block data, amount, tokenId, fee is already changed to send block data by generator
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(blockContext, sendBlock, quota.CalcCreateQuota(sendBlock.Fee))
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(blockContext, sendBlock)
		}
	case ledger.BlockTypeSendCreate:
		quotaTotal, quotaAddition := nodeConfig.calcQuota(database, block.AccountAddress, quota.IsPoW(block.Nonce))
		blockContext, err = vm.sendCreate(blockContext, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return []*vm_context.VmAccountBlock{blockContext}, NoRetry, nil
		}
	case ledger.BlockTypeSendCall:
		quotaTotal, quotaAddition := nodeConfig.calcQuota(database, block.AccountAddress, quota.IsPoW(block.Nonce))
		blockContext, err = vm.sendCall(blockContext, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return []*vm_context.VmAccountBlock{blockContext}, NoRetry, nil
		}
	}
	return nil, NoRetry, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	defer monitor.LogTime("vm", "SendCreate", time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := quota.IntrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = quota.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}

	contractFee, err := calcContractFee(block.AccountBlock.Data)
	if err != nil {
		return nil, err
	}

	gid := contracts.GetGidFromCreateContractData(block.AccountBlock.Data)
	if !isExistGid(block.VmContext, gid) {
		return nil, errors.New("consensus group not exist")
	}

	if !CanTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
		return nil, ErrInsufficientBalance
	}

	contractAddr := contracts.NewContractAddress(
		block.AccountBlock.AccountAddress,
		block.AccountBlock.Height,
		block.AccountBlock.PrevHash,
		block.AccountBlock.SnapshotHash)
	if block.VmContext.IsAddressExisted(&contractAddr) {
		return nil, ErrContractAddressCreationFail
	}

	block.AccountBlock.Fee = contractFee
	block.AccountBlock.ToAddress = contractAddr
	// sub balance and service fee
	block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	if block.AccountBlock.Fee != nil {
		block.VmContext.SubBalance(&ledger.ViteTokenId, block.AccountBlock.Fee)
	}
	vm.updateBlock(block, nil, quota.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	block.VmContext.SetContractGid(&gid, &contractAddr)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock, quotaTotal uint64) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	defer monitor.LogTime("vm", "ReceiveCreate", time.Now())
	quotaLeft := quotaTotal
	if block.VmContext.IsAddressExisted(&block.AccountBlock.AccountAddress) {
		return nil, NoRetry, ErrAddressCollision
	}
	// check can make transaction
	cost, err := quota.IntrinsicGasCost(nil, true)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft, err = quota.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, NoRetry, err
	}

	vm.blockList = []*vm_context.VmAccountBlock{block}

	// create contract account and add balance
	block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)

	sendBlock.Data = sendBlock.Data[types.GidSize:]

	// init contract state and set contract code
	c := newContract(sendBlock.AccountAddress, block.AccountBlock.AccountAddress, block, sendBlock, quotaLeft, 0)
	c.setCallCode(block.AccountBlock.AccountAddress, sendBlock.Data)
	code, err := c.run(vm)
	if err == nil && len(code) <= MaxCodeSize {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = quota.UseQuota(c.quotaLeft, codeCost)
		if err == nil {
			block.VmContext.SetContractCode(code)
			block.AccountBlock.Data = block.VmContext.GetStorageHash().Bytes()
			vm.updateBlock(block, nil, 0)
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
	defer monitor.LogTime("vm", "SendCall", time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok, err := getPrecompiledContract(block.AccountBlock.ToAddress, block.AccountBlock.Data); ok {
		if err != nil {
			return nil, err
		}
		block.AccountBlock.Fee, err = p.getFee(vm, block)
		if err != nil {
			return nil, err
		}
		if !CanTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
			return nil, ErrInsufficientBalance
		}
		quotaLeft, err = p.doSend(vm, block, quotaLeft)
		if err != nil {
			return nil, err
		}
		block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
		block.VmContext.SubBalance(&ledger.ViteTokenId, block.AccountBlock.Fee)
	} else {
		block.AccountBlock.Fee = helper.Big0
		cost, err := quota.IntrinsicGasCost(block.AccountBlock.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = quota.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
		if !CanTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
			return nil, ErrInsufficientBalance
		}
		block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	}
	var quotaUsed uint64
	if isPrecompiledContractAddress(block.AccountBlock.AccountAddress) {
		quotaUsed = 0
	} else {
		quotaUsed = quota.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, 0, nil)
	}
	vm.updateBlock(block, nil, quotaUsed)
	return block, nil

}

func (vm *VM) receiveCall(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	defer monitor.LogTime("vm", "ReceiveCall", time.Now())
	if p, ok, _ := getPrecompiledContract(block.AccountBlock.AccountAddress, sendBlock.Data); ok {
		vm.blockList = []*vm_context.VmAccountBlock{block}
		block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		err := p.doReceive(vm, block, sendBlock)
		if err == nil {
			block.AccountBlock.Data = block.VmContext.GetStorageHash().Bytes()
			vm.updateBlock(block, err, 0)
			if err = vm.doSendBlockList(quota.TxGas); err == nil {
				return vm.blockList, NoRetry, nil
			}
		}
		vm.revert(block)
		block.AccountBlock.Data = nil
		vm.updateBlock(block, err, 0)
		return vm.blockList, NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition := nodeConfig.calcQuota(block.VmContext, block.AccountBlock.AccountAddress, quota.IsPoW(block.AccountBlock.Nonce))
		quotaLeft := quotaTotal
		quotaRefund := uint64(0)
		cost, err := quota.IntrinsicGasCost(nil, false)
		if err != nil {
			return nil, NoRetry, err
		}
		quotaLeft, err = quota.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, Retry, err
		}
		vm.blockList = []*vm_context.VmAccountBlock{block}
		// add balance, create account if not exist
		block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		// do transfer transaction if account code size is zero
		code := block.VmContext.GetContractCode(&block.AccountBlock.AccountAddress)
		if len(code) == 0 {
			vm.updateBlock(block, nil, quota.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
			return vm.blockList, NoRetry, nil
		}
		// run code
		c := newContract(sendBlock.AccountAddress, block.AccountBlock.AccountAddress, block, sendBlock, quotaLeft, quotaRefund)
		c.setCallCode(block.AccountBlock.AccountAddress, code)
		_, err = c.run(vm)
		if err == nil {
			block.AccountBlock.Data = block.VmContext.GetStorageHash().Bytes()
			vm.updateBlock(block, nil, quota.CalcQuotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil))
			err = vm.doSendBlockList(quotaTotal - quotaAddition - block.AccountBlock.Quota)
			if err == nil {
				return vm.blockList, NoRetry, nil
			}
		}

		vm.revert(block)
		block.AccountBlock.Data = nil
		vm.updateBlock(block, err, quota.CalcQuotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err))
		return vm.blockList, err == quota.ErrOutOfQuota, err
	}
}

func (vm *VM) sendReward(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	defer monitor.LogTime("vm", "SendReward", time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	cost, err := quota.IntrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = quota.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if block.AccountBlock.AccountAddress != contracts.AddressRegister &&
		block.AccountBlock.AccountAddress != contracts.AddressMintage {
		return nil, errors.New("invalid account address")
	}
	vm.updateBlock(block, nil, 0)
	return block, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	code := c.block.VmContext.GetContractCode(&contractAddr)
	if len(code) > 0 {
		cNew := newContract(c.caller, c.address, c.block, c.sendBlock, c.quotaLeft, c.quotaRefund)
		cNew.setCallCode(contractAddr, code)
		ret, err = cNew.run(vm)
		c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
		return ret, err
	}
	return nil, nil
}

func (vm *VM) updateBlock(block *vm_context.VmAccountBlock, err error, quotaUsed uint64) {
	block.AccountBlock.Quota = quotaUsed
	block.AccountBlock.StateHash = *block.VmContext.GetStorageHash()
	if block.AccountBlock.IsReceiveBlock() {
		block.AccountBlock.LogHash = block.VmContext.GetLogListHash()
		if err == quota.ErrOutOfQuota {
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
	block.VmContext.Reset()
}

func CanTransfer(db vmctxt_interface.VmDatabase, addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
	if feeAmount.Sign() == 0 {
		return tokenAmount.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0
	}
	if IsViteToken(tokenTypeId) {
		balance := new(big.Int).Add(tokenAmount, feeAmount)
		return balance.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0
	}
	return tokenAmount.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0 && feeAmount.Cmp(db.GetBalance(&addr, &ledger.ViteTokenId)) <= 0
}

func (vm *VM) getNewBlockHeight(block *vm_context.VmAccountBlock) uint64 {
	return block.AccountBlock.Height + uint64(len(vm.blockList))
}

func calcContractFee(data []byte) (*big.Int, error) {
	return createContractFee, nil
}

func isExistGid(db vmctxt_interface.VmDatabase, gid types.Gid) bool {
	value := db.GetStorage(&contracts.AddressConsensusGroup, contracts.GetConsensusGroupKey(gid))
	return len(value) > 0
}

func makeSendBlock(block *ledger.AccountBlock, toAddress types.Address, blockType byte, amount *big.Int, tokenId types.TokenTypeId, height uint64, data []byte) *ledger.AccountBlock {
	newTimestamp := time.Unix(0, block.Timestamp.UnixNano())
	return &ledger.AccountBlock{
		AccountAddress: block.AccountAddress,
		ToAddress:      toAddress,
		BlockType:      blockType,
		Amount:         amount,
		TokenId:        tokenId,
		Height:         height,
		SnapshotHash:   block.SnapshotHash,
		Data:           data,
		Fee:            big.NewInt(0),
		Timestamp:      &newTimestamp,
	}
}
