package vm

import (
	"bytes"
	"github.com/modern-go/reflect2"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

// memoryGasCosts calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.
func memoryGasCost(mem *memory, newMemSize uint64) (uint64, bool, error) {

	if newMemSize == 0 {
		return 0, true, nil
	}
	// The maximum that will fit in a uint64 is max_word_count - 1
	// anything above that will result in an overflow.
	// Additionally, a newMemSize which results in a
	// newMemSizeWords larger than 0x7ffffffff will cause the square operation
	// to overflow.
	// The constant ç is the highest number that can be used without
	// overflowing the gas calculation
	if newMemSize > 0xffffffffe0 {
		return 0, true, util.ErrGasUintOverflow
	}

	newMemSizeWords := helper.ToWordSize(newMemSize)
	newMemSize = newMemSizeWords * helper.WordSize

	if newMemSize > uint64(mem.len()) {
		square := newMemSizeWords * newMemSizeWords
		linCoef := newMemSizeWords * memoryGas
		quadCoef := square / quadCoeffDiv
		newTotalFee := linCoef + quadCoef

		fee := newTotalFee - mem.lastGasCost
		mem.lastGasCost = newTotalFee

		return fee, true, nil
	}
	return 0, true, nil
}

func constGasFunc(gas uint64) gasFunc {
	return func(vm *VM, contrac *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
		return gas, true, nil
	}
}

func gasExp(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	expByteLen := uint64((stack.back(1).BitLen() + 7) / 8)

	var (
		gas      = expByteLen * expByteGas // no overflow check required. Max is 256 * expByteGas gas
		overflow bool
	)
	if gas, overflow = helper.SafeAdd(gas, slowStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasBlake2b(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	if gas, overflow = helper.SafeAdd(gas, blake2bGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(1))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), blake2bWordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasCallDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	words, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if words, overflow = helper.SafeMul(helper.ToWordSize(words), copyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, words); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), copyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasExtCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, extCodeCopyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(3))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), copyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasReturnDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	words, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if words, overflow = helper.SafeMul(helper.ToWordSize(words), copyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, words); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasMLoad(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasMStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasMStore8(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, fastestStepGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasSStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var (
		newValue   = stack.back(1)
		loc        = stack.back(0)
		locHash, _ = types.BigToHash(loc)
	)
	currentValue, err := c.db.GetValue(locHash.Bytes())
	util.DealWithErr(err)
	if bytes.Equal(currentValue, newValue.Bytes()) {
		return sstoreNoopGas, true, nil
	}
	originalValue, err := c.db.GetOriginalValue(locHash.Bytes())
	util.DealWithErr(err)
	if bytes.Equal(originalValue, currentValue) {
		if len(originalValue) == 0 {
			return sstoreInitGas, true, nil
		}
		if newValue.Sign() == 0 {
			return sstoreCleanGas, true, nil
		}
		return sstoreResetGas, true, nil
	}
	// value changed again, charge 200 for first change
	if bytes.Equal(originalValue, newValue.Bytes()) {
		if len(originalValue) == 0 {
			return sstoreInitGas - sstoreMemoryGas - sstoreNoopGas, false, nil
		}
		if len(currentValue) == 0 {
			return sstoreNoopGas + sstoreMemoryGas - sstoreCleanGas, true, nil
		}
		return sstoreResetGas - sstoreMemoryGas - sstoreNoopGas, false, nil
	}
	if len(originalValue) > 0 {
		if len(currentValue) == 0 && newValue.Sign() > 0 {
			return sstoreResetGas + sstoreMemoryGas - sstoreCleanGas, true, nil
		}
		if len(currentValue) > 0 && newValue.Sign() == 0 {
			return sstoreResetGas - sstoreMemoryGas - sstoreCleanGas, false, nil
		}
	}
	return sstoreMemoryGas, true, nil
}

func gasPush(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return fastestStepGas, true, nil
}

func gasDup(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return fastestStepGas, true, nil
}

func gasSwap(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return fastestStepGas, true, nil
}

func makeGasLog(n uint64) gasFunc {
	return func(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
		requestedSize, overflow := helper.BigUint64(stack.back(1))
		if overflow {
			return 0, true, util.ErrGasUintOverflow
		}

		gas, _, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, true, err
		}

		if gas, overflow = helper.SafeAdd(gas, logGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
		if gas, overflow = helper.SafeAdd(gas, n*logTopicGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = helper.SafeMul(requestedSize, logDataGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
		if gas, overflow = helper.SafeAdd(gas, memorySizeGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
		return gas, true, nil
	}
}

func gasDelegateCall(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}
	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, delegateCallGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasCall(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, true, err
	}
	toAddrBig, tokenIdBig, amount, inOffset, inSize := stack.back(0), stack.back(1), stack.back(2), stack.back(3), stack.back(4)
	toAddress, _ := types.BigToAddress(toAddrBig)
	tokenId, _ := types.BigToTokenTypeId(tokenIdBig)
	cost, err := GasRequiredForSendBlock(util.MakeSendBlock(
		c.block.AccountAddress,
		toAddress,
		ledger.BlockTypeSendCall,
		amount,
		tokenId,
		mem.get(inOffset.Int64(), inSize.Int64())))
	if err != nil {
		return 0, true, err
	}
	quotaRatio, err := getQuotaRatioForRS(c.db, toAddress, c.sendBlock, vm.globalStatus)
	if err != nil {
		return 0, true, err
	}
	cost, err = util.MultipleCost(cost, quotaRatio)
	if err != nil {
		return 0, true, err
	}

	if cost > callMinusGas {
		cost = cost - callMinusGas
		var overflow bool
		if gas, overflow = helper.SafeAdd(gas, cost); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
	}
	return gas, true, nil
}

func gasReturn(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return memoryGasCost(mem, memorySize)
}

func gasRevert(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return memoryGasCost(mem, memorySize)
}

func GasRequiredForBlock(db vm_db.VmDb, block *ledger.AccountBlock) (uint64, error) {
	if block.BlockType == ledger.BlockTypeReceive {
		return gasReceive(block, nil)
	} else {
		cost, err := GasRequiredForSendBlock(block)
		if err != nil {
			return 0, err
		}
		if block.BlockType != ledger.BlockTypeSendCall {
			return cost, nil
		}
		quotaRatio, err := getQuotaRatioForS(db, block.ToAddress)
		if err != nil {
			return 0, err
		}
		cost, err = util.MultipleCost(cost, quotaRatio)
		if err != nil {
			return 0, err
		}
		return cost, nil
	}
}

func GasRequiredForSendBlock(block *ledger.AccountBlock) (uint64, error) {
	if block.BlockType == ledger.BlockTypeSendCreate {
		return gasNormalSendCall(block)
	} else if block.BlockType == ledger.BlockTypeSendCall {
		return gasUserSendCall(block)
	} else {
		return 0, util.ErrBlockTypeNotSupported
	}
}

func gasReceiveCreate(block *ledger.AccountBlock, meta *ledger.ContractMeta) (uint64, error) {
	confirmTime := uint8(0)
	if meta != nil {
		confirmTime = meta.SendConfirmedTimes
	}
	return util.IntrinsicGasCost(nil, true, confirmTime)
}

func gasReceive(block *ledger.AccountBlock, meta *ledger.ContractMeta) (uint64, error) {
	confirmTime := uint8(0)
	if meta != nil {
		confirmTime = meta.SendConfirmedTimes
	}
	return util.IntrinsicGasCost(nil, false, confirmTime)
}

func gasUserSendCall(block *ledger.AccountBlock) (uint64, error) {
	if types.IsBuiltinContractAddrInUse(block.ToAddress) {
		if method, ok, err := contracts.GetBuiltinContract(block.ToAddress, block.Data); !ok || err != nil {
			return 0, util.ErrAbiMethodNotFound
		} else {
			return method.GetSendQuota(block.Data)
		}
	} else {
		return gasNormalSendCall(block)
	}
}
func gasNormalSendCall(block *ledger.AccountBlock) (uint64, error) {
	return util.IntrinsicGasCost(block.Data, false, 0)
}

// For normal send block:
// 1. toAddr is user, quota ratio is 1;
// 2. toAddr is contract, contract is created in latest snapshot block, return quota ratio
// 3. toAddr is contract, contract is not created in latest snapshot block, return error
func getQuotaRatioForS(db vm_db.VmDb, toAddr types.Address) (uint8, error) {
	if !types.IsContractAddr(toAddr) {
		return util.CommonQuotaRatio, nil
	}
	sb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	return getQuotaRatioBySnapshotBlock(db, toAddr, sb)
}

// For send block generated by contract receive block:
// 1. toAddr is user, quota ratio is 1;
// 2. toAddr is contract, send block is confirmed, contract is created in confirm status, return quota ratio
// 3. toAddr is contract, send block is confirmed, contract is not created in confirm status, return error
// 4. toAddr is contract, send block is not confirmed, contract is created in latest block, return quota ratio
// 5. toAddr is contract, send block is not confirmed, contract is not created in latest block, wait for a reliable status
func getQuotaRatioForRS(db vm_db.VmDb, toAddr types.Address, sendBlock *ledger.AccountBlock, status util.GlobalStatus) (uint8, error) {
	if !types.IsContractAddr(toAddr) {
		return util.CommonQuotaRatio, nil
	}
	if !reflect2.IsNil(status) && status.SnapshotBlock() != nil {
		return getQuotaRatioBySnapshotBlock(db, toAddr, status.SnapshotBlock())
	}
	confirmSb, err := db.GetConfirmSnapshotHeader(sendBlock.Hash)
	util.DealWithErr(err)
	if confirmSb != nil {
		return getQuotaRatioBySnapshotBlock(db, toAddr, confirmSb)
	}
	sb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	meta, err := db.GetContractMetaInSnapshot(toAddr, sb)
	util.DealWithErr(err)
	if meta != nil {
		return meta.QuotaRatio, nil
	}
	return 0, util.ErrNoReliableStatus
}

func getQuotaRatioBySnapshotBlock(db vm_db.VmDb, toAddr types.Address, snapshotBlock *ledger.SnapshotBlock) (uint8, error) {
	meta, err := db.GetContractMetaInSnapshot(toAddr, snapshotBlock)
	util.DealWithErr(err)
	if meta == nil {
		return 0, util.ErrContractNotExists
	}
	return meta.QuotaRatio, nil
}
