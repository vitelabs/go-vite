package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

// memoryGasCosts calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.
func memoryGasCost(vm *VM, mem *memory, newMemSize uint64) (uint64, bool, error) {

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
		linCoef := newMemSizeWords * vm.gasTable.MemGas
		quadCoef := square / vm.gasTable.MemGasDivision
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

func gasAdd(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.AddGas, true, nil
}
func gasMul(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.MulGas, true, nil
}
func gasSub(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SubGas, true, nil
}
func gasDiv(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.DivGas, true, nil
}
func gasSdiv(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SdivGas, true, nil
}
func gasMod(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.ModGas, true, nil
}
func gasSmod(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SmodGas, true, nil
}
func gasAddmod(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.AddmodGas, true, nil
}
func gasMulmod(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.MulmodGas, true, nil
}
func gasSignextend(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SignextendGas, true, nil
}
func gasLt(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.LtGas, true, nil
}
func gasGt(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.GtGas, true, nil
}
func gasSlt(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SltGas, true, nil
}
func gasSgt(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SgtGas, true, nil
}
func gasEq(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.EqGas, true, nil
}
func gasIszero(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.IszeroGas, true, nil
}
func gasAnd(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.AndGas, true, nil
}
func gasOr(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.OrGas, true, nil
}
func gasXor(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.XorGas, true, nil
}
func gasNot(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.NotGas, true, nil
}
func gasByte(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.ByteGas, true, nil
}
func gasShl(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.ShlGas, true, nil
}
func gasShr(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.ShrGas, true, nil
}
func gasSar(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SarGas, true, nil
}
func gasAddress(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.AddressGas, true, nil
}
func gasBalance(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.BalanceGas, true, nil
}
func gasCaller(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.CallerGas, true, nil
}
func gasCallvalue(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.CallvalueGas, true, nil
}
func gasCalldataload(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.CalldataloadGas, true, nil
}
func gasCalldatasize(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.CalldatasizeGas, true, nil
}
func gasCodesize(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.CodesizeGas, true, nil
}
func gasReturndatasize(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.ReturndatasizeGas, true, nil
}
func gasTimestamp(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.TimestampGas, true, nil
}
func gasHeight(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.HeightGas, true, nil
}
func gasTokenid(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.TokenidGas, true, nil
}
func gasAccountheight(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.AccountheightGas, true, nil
}
func gasPrevhash(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.PrevhashGas, true, nil
}
func gasFromhash(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.FromhashGas, true, nil
}
func gasSeed(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SeedGas, true, nil
}
func gasRandom(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.RandomGas, true, nil
}
func gasPop(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.PopGas, true, nil
}
func gasSload(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SloadGas, true, nil
}
func gasJump(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.JumpGas, true, nil
}
func gasJumpi(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.JumpiGas, true, nil
}
func gasPc(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.PcGas, true, nil
}
func gasMsize(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.MsizeGas, true, nil
}
func gasJumpdest(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.JumpdestGas, true, nil
}

func gasExp(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	expByteLen := uint64((stack.back(1).BitLen() + 7) / 8)

	var (
		gas      = expByteLen * vm.gasTable.ExpByteGas // no overflow check required. Max is 256 * expByteGas gas
		overflow bool
	)
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.ExpGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasBlake2b(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.Blake2bGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(1))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), vm.gasTable.Blake2bWordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasCallDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.CalldatacopyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	words, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if words, overflow = helper.SafeMul(helper.ToWordSize(words), vm.gasTable.MemcopyWordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, words); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasCodeCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.CodeCopyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	wordGas, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if wordGas, overflow = helper.SafeMul(helper.ToWordSize(wordGas), vm.gasTable.MemcopyWordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, wordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasReturnDataCopy(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, err
	}

	var overflow bool
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.ReturndatacopyGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	words, overflow := helper.BigUint64(stack.back(2))
	if overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if words, overflow = helper.SafeMul(helper.ToWordSize(words), vm.gasTable.MemcopyWordGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}

	if gas, overflow = helper.SafeAdd(gas, words); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasMLoad(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.MloadGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasMStore(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.MstoreGas); overflow {
		return 0, true, util.ErrGasUintOverflow
	}
	return gas, true, nil
}

func gasMStore8(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	var overflow bool
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, util.ErrGasUintOverflow
	}
	if gas, overflow = helper.SafeAdd(gas, vm.gasTable.Mstore8Gas); overflow {
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

	currentValue := util.GetValue(c.db, locHash.Bytes())
	if bytes.Equal(currentValue, newValue.Bytes()) {
		return vm.gasTable.SstoreNoopGas, true, nil
	}
	originalValue, err := c.db.GetOriginalValue(locHash.Bytes())
	util.DealWithErr(err)
	if bytes.Equal(originalValue, currentValue) {
		if len(originalValue) == 0 {
			return vm.gasTable.SstoreInitGas, true, nil
		}
		if newValue.Sign() == 0 {
			return vm.gasTable.SstoreCleanGas, true, nil
		}
		return vm.gasTable.SstoreResetGas, true, nil
	}
	// value changed again, charge 200 for first change
	if bytes.Equal(originalValue, newValue.Bytes()) {
		if len(originalValue) == 0 {
			return vm.gasTable.SstoreInitGas - vm.gasTable.SstoreMemGas - vm.gasTable.SstoreNoopGas, false, nil
		}
		if len(currentValue) == 0 {
			return vm.gasTable.SstoreNoopGas + vm.gasTable.SstoreMemGas - vm.gasTable.SstoreCleanGas, true, nil
		}
		return vm.gasTable.SstoreResetGas - vm.gasTable.SstoreMemGas - vm.gasTable.SstoreNoopGas, false, nil
	}
	if len(originalValue) > 0 {
		if len(currentValue) == 0 && newValue.Sign() > 0 {
			return vm.gasTable.SstoreResetGas + vm.gasTable.SstoreMemGas - vm.gasTable.SstoreCleanGas, true, nil
		}
		if len(currentValue) > 0 && newValue.Sign() == 0 {
			return vm.gasTable.SstoreResetGas - vm.gasTable.SstoreMemGas - vm.gasTable.SstoreCleanGas, false, nil
		}
	}
	return vm.gasTable.SstoreMemGas, true, nil
}

func gasPush(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.PushGas, true, nil
}

func gasDup(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.DupGas, true, nil
}

func gasSwap(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return vm.gasTable.SwapGas, true, nil
}

func makeGasLog(n uint64) gasFunc {
	return func(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
		requestedSize, overflow := helper.BigUint64(stack.back(1))
		if overflow {
			return 0, true, util.ErrGasUintOverflow
		}

		gas, _, err := memoryGasCost(vm, mem, memorySize)
		if err != nil {
			return 0, true, err
		}

		if gas, overflow = helper.SafeAdd(gas, vm.gasTable.LogGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
		if gas, overflow = helper.SafeAdd(gas, n*vm.gasTable.LogTopicGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = helper.SafeMul(requestedSize, vm.gasTable.LogDataGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
		if gas, overflow = helper.SafeAdd(gas, memorySizeGas); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
		return gas, true, nil
	}
}

func gasCall(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	gas, _, err := memoryGasCost(vm, mem, memorySize)
	if err != nil {
		return 0, true, err
	}
	toAddrBig, tokenIDBig, amount, inOffset, inSize := stack.back(0), stack.back(1), stack.back(2), stack.back(3), stack.back(4)
	toAddress, _ := types.BigToAddress(toAddrBig)
	tokenID, _ := types.BigToTokenTypeId(tokenIDBig)
	cost, err := gasRequiredForSendBlock(
		util.MakeSendBlock(
			c.block.AccountAddress,
			toAddress,
			ledger.BlockTypeSendCall,
			amount,
			tokenID,
			mem.get(inOffset.Int64(), inSize.Int64())),
		vm.gasTable,
		vm.latestSnapshotHeight)
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

	if cost > vm.gasTable.CallMinusGas {
		cost = cost - vm.gasTable.CallMinusGas
		var overflow bool
		if gas, overflow = helper.SafeAdd(gas, cost); overflow {
			return 0, true, util.ErrGasUintOverflow
		}
	}
	return gas, true, nil
}

func gasReturn(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return memoryGasCost(vm, mem, memorySize)
}

func gasRevert(vm *VM, c *contract, stack *stack, mem *memory, memorySize uint64) (uint64, bool, error) {
	return memoryGasCost(vm, mem, memorySize)
}

// GasRequiredForBlock calculates gas required for a user account block.
func GasRequiredForBlock(db vm_db.VmDb, block *ledger.AccountBlock, gasTable *util.GasTable, sbHeight uint64) (uint64, error) {
	if block.BlockType == ledger.BlockTypeReceive {
		return gasReceive(block, nil, gasTable)
	}
	cost, err := gasRequiredForSendBlock(block, gasTable, sbHeight)
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

func gasRequiredForSendBlock(block *ledger.AccountBlock, gasTable *util.GasTable, sbHeight uint64) (uint64, error) {
	if block.BlockType == ledger.BlockTypeSendCreate {
		return gasSendCreate(block, gasTable)
	} else if block.BlockType == ledger.BlockTypeSendCall {
		return gasUserSendCall(block, gasTable, sbHeight)
	} else {
		return 0, util.ErrBlockTypeNotSupported
	}
}

func gasSendCreate(block *ledger.AccountBlock, gasTable *util.GasTable) (uint64, error) {
	return util.IntrinsicGasCost(block.Data, gasTable.CreateTxRequestGas, 0, gasTable)
}

func gasReceiveCreate(block *ledger.AccountBlock, meta *ledger.ContractMeta, gasTable *util.GasTable) (uint64, error) {
	confirmTime := uint8(0)
	if meta != nil {
		confirmTime = meta.SendConfirmedTimes
	}
	return util.IntrinsicGasCost(nil, gasTable.CreateTxResponseGas, confirmTime, gasTable)
}

func gasUserSendCall(block *ledger.AccountBlock, gasTable *util.GasTable, sbHeight uint64) (uint64, error) {
	if types.IsBuiltinContractAddrInUse(block.ToAddress) {
		method, ok, err := contracts.GetBuiltinContractMethod(block.ToAddress, block.Data, sbHeight)
		if !ok || err != nil {
			return 0, util.ErrAbiMethodNotFound
		}
		return method.GetSendQuota(block.Data, gasTable)
	}
	return gasSendCall(block, gasTable)
}

func gasReceive(block *ledger.AccountBlock, meta *ledger.ContractMeta, gasTable *util.GasTable) (uint64, error) {
	confirmTime := uint8(0)
	if meta != nil {
		confirmTime = meta.SendConfirmedTimes
	}
	return util.IntrinsicGasCost(nil, gasTable.TxGas, confirmTime, gasTable)
}

func gasSendCall(block *ledger.AccountBlock, gasTable *util.GasTable) (uint64, error) {
	return util.IntrinsicGasCost(block.Data, gasTable.TxGas, 0, gasTable)
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
	if !helper.IsNil(status) && status.SnapshotBlock() != nil {
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
