/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"encoding/hex"
	"errors"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/fork"

	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type VMConfig struct {
	Debug bool
}

type NodeConfig struct {
	isTest      bool
	calcQuota   func(db vmctxt_interface.VmDatabase, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error)
	canTransfer func(db vmctxt_interface.VmDatabase, addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool

	interpreterLog log15.Logger
	log            log15.Logger
	IsDebug        bool
}

var nodeConfig NodeConfig

func IsTest() bool {
	return nodeConfig.isTest
}

func InitVmConfig(isTest bool, isTestParam bool, isDebug bool, datadir string) {
	if isTest {
		nodeConfig = NodeConfig{
			isTest: isTest,
			calcQuota: func(db vmctxt_interface.VmDatabase, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
				return 1000000, 0, nil
			},
			canTransfer: func(db vmctxt_interface.VmDatabase, addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
				return true
			},
		}
	} else {
		nodeConfig = NodeConfig{
			isTest: isTest,
			calcQuota: func(db vmctxt_interface.VmDatabase, addr types.Address, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
				return quota.CalcQuota(db, addr, pledgeAmount, difficulty)
			},
			canTransfer: func(db vmctxt_interface.VmDatabase, addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
				if feeAmount.Sign() == 0 {
					return tokenAmount.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0
				}
				if util.IsViteToken(tokenTypeId) {
					balance := new(big.Int).Add(tokenAmount, feeAmount)
					return balance.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0
				}
				return tokenAmount.Cmp(db.GetBalance(&addr, &tokenTypeId)) <= 0 && feeAmount.Cmp(db.GetBalance(&addr, &ledger.ViteTokenId)) <= 0
			},
		}
	}
	nodeConfig.log = log15.New("module", "vm")
	nodeConfig.interpreterLog = log15.New("module", "vm")
	contracts.InitContractsConfig(isTestParam)
	quota.InitQuotaConfig(isTestParam)
	nodeConfig.IsDebug = isDebug
	if isDebug {
		InitLog(datadir, "dbug")
	}
}

func InitLog(dir, lvl string) {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	path := filepath.Join(dir, "vmlog", time.Now().Format("2006-01-02T15-04"))
	filename := filepath.Join(path, "vm.log")
	nodeConfig.log.SetHandler(
		log15.LvlFilterHandler(logLevel, log15.StreamHandler(common.MakeDefaultLogger(filename), log15.LogfmtFormat())),
	)
	interpreterFileName := filepath.Join(path, "interpreter.log")
	nodeConfig.interpreterLog.SetHandler(
		log15.LvlFilterHandler(logLevel, log15.StreamHandler(common.MakeDefaultLogger(interpreterFileName), log15.LogfmtFormat())),
	)
}

type VmContext struct {
	blockList []*vm_context.VmAccountBlock
}

type VM struct {
	VMConfig
	abort int32
	VmContext
	i *Interpreter
}

func NewVM() *VM {
	return &VM{}
}

func printDebugBlockInfo(block *ledger.AccountBlock, blockList []*vm_context.VmAccountBlock, err error) {
	responseBlockList := make([]string, 0)
	if len(blockList) > 0 {
		for _, b := range blockList[:] {
			if b.AccountBlock.IsSendBlock() {
				responseBlockList = append(responseBlockList,
					"{SelfAddr: "+b.AccountBlock.AccountAddress.String()+", "+
						"ToAddr: "+b.AccountBlock.ToAddress.String()+", "+
						"BlockType: "+strconv.FormatInt(int64(b.AccountBlock.BlockType), 10)+", "+
						"Quota: "+strconv.FormatUint(b.AccountBlock.Quota, 10)+", "+
						"Amount: "+b.AccountBlock.Amount.String()+", "+
						"TokenId: "+b.AccountBlock.TokenId.String()+", "+
						"Height: "+strconv.FormatUint(b.AccountBlock.Height, 10)+", "+
						"Data: "+hex.EncodeToString(b.AccountBlock.Data)+", "+
						"Fee: "+b.AccountBlock.Fee.String()+"}")
			} else {
				responseBlockList = append(responseBlockList,
					"{SelfAddr: "+b.AccountBlock.AccountAddress.String()+", "+
						"FromHash: "+b.AccountBlock.FromBlockHash.String()+", "+
						"BlockType: "+strconv.FormatInt(int64(b.AccountBlock.BlockType), 10)+", "+
						"Quota: "+strconv.FormatUint(b.AccountBlock.Quota, 10)+", "+
						"Height: "+strconv.FormatUint(b.AccountBlock.Height, 10)+", "+
						"Data: "+hex.EncodeToString(b.AccountBlock.Data)+"}")
			}
		}
	}
	nodeConfig.log.Info("vm run stop",
		"blockType", block.BlockType,
		"address", block.AccountAddress.String(),
		"height", block.Height,
		"fromHash", block.FromBlockHash.String(),
		"err", err,
		"generatedBlockList", responseBlockList,
	)
}

func (vm *VM) Run(database vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	defer monitor.LogTime("vm", "run", time.Now())
	defer func() {
		if nodeConfig.IsDebug {
			printDebugBlockInfo(block, blockList, err)
		}
	}()
	if nodeConfig.IsDebug {
		nodeConfig.log.Info("vm run start",
			"blockType", block.BlockType,
			"address", block.AccountAddress.String(),
			"height", block.Height, ""+
				"fromHash", block.FromBlockHash.String())
	}
	blockContext := &vm_context.VmAccountBlock{block.Copy(), database}
	vm.i = NewInterpreter(database.CurrentSnapshotBlock().Height)
	switch block.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		blockContext.AccountBlock.Data = nil
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			if !fork.IsSmartFork(database.CurrentSnapshotBlock().Height) {
				return nil, NoRetry, errors.New("snapshot height not supported")
			}
			return vm.receiveCreate(blockContext, sendBlock, quota.CalcCreateQuota(sendBlock.Fee))
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(blockContext, sendBlock)
		} else if sendBlock.BlockType == ledger.BlockTypeSendRefund {
			return vm.receiveRefund(blockContext, sendBlock)
		}
	case ledger.BlockTypeSendCreate:
		if !fork.IsSmartFork(database.CurrentSnapshotBlock().Height) {
			return nil, NoRetry, errors.New("snapshot height not supported")
		}
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			database,
			block.AccountAddress,
			abi.GetPledgeBeneficialAmount(database, block.AccountAddress),
			block.Difficulty)
		if err != nil {
			return nil, NoRetry, err
		}
		blockContext, err = vm.sendCreate(blockContext, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return []*vm_context.VmAccountBlock{blockContext}, NoRetry, nil
		}
	case ledger.BlockTypeSendCall:
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			database,
			block.AccountAddress,
			abi.GetPledgeBeneficialAmount(database, block.AccountAddress),
			block.Difficulty)
		if err != nil {
			return nil, NoRetry, err
		}
		blockContext, err = vm.sendCall(blockContext, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return []*vm_context.VmAccountBlock{blockContext}, NoRetry, nil
		}
	case ledger.BlockTypeSendReward, ledger.BlockTypeSendRefund:
		return nil, NoRetry, util.ErrContractSendBlockRunFailed
	}
	return nil, NoRetry, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	//defer monitor.LogTime("vm", "SendCreate", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "sendCreate")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	// check can make transaction
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := util.IntrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}

	block.AccountBlock.Fee, err = calcContractFee(block.AccountBlock.Data)
	if err != nil {
		return nil, err
	}

	gid := util.GetGidFromCreateContractData(block.AccountBlock.Data)
	if gid == types.SNAPSHOT_GID {
		return nil, errors.New("invalid consensus group")
	}
	if !contracts.IsExistGid(block.VmContext, gid) {
		return nil, errors.New("consensus group not exist")
	}

	contractType := util.GetContractTypeFromCreateContractData(block.AccountBlock.Data)
	if !util.IsExistContractType(contractType) {
		return nil, errors.New("invalid contract type")
	}

	if !nodeConfig.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
		return nil, util.ErrInsufficientBalance
	}

	contractAddr := util.NewContractAddress(
		block.AccountBlock.AccountAddress,
		block.AccountBlock.Height,
		block.AccountBlock.PrevHash,
		block.AccountBlock.SnapshotHash)
	if block.VmContext.IsAddressExisted(&contractAddr) {
		return nil, util.ErrContractAddressCreationFail
	}

	block.AccountBlock.ToAddress = contractAddr
	// sub balance and service fee
	block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	block.VmContext.SubBalance(&ledger.ViteTokenId, block.AccountBlock.Fee)
	vm.updateBlock(block, nil, util.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	block.VmContext.SetContractGid(&gid, &contractAddr)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock, quotaTotal uint64) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	//defer monitor.LogTime("vm", "ReceiveCreate", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "receiveCreate")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	quotaLeft := quotaTotal
	if block.VmContext.IsAddressExisted(&block.AccountBlock.AccountAddress) {
		return nil, NoRetry, util.ErrAddressCollision
	}
	// check can make transaction
	cost, err := util.IntrinsicGasCost(nil, true)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, NoRetry, err
	}

	vm.blockList = []*vm_context.VmAccountBlock{block}

	// create contract account and add balance
	block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)

	// init contract state and set contract code
	initCode := util.GetCodeFromCreateContractData(sendBlock.Data)
	c := newContract(block, sendBlock, initCode, quotaLeft, 0)
	c.setCallCode(block.AccountBlock.AccountAddress, initCode)
	code, err := c.run(vm)
	if err == nil && len(code) <= MaxCodeSize {
		code := util.PackContractCode(util.GetContractTypeFromCreateContractData(sendBlock.Data), code)
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = util.UseQuota(c.quotaLeft, codeCost)
		if err == nil {
			block.VmContext.SetContractCode(code)
			block.AccountBlock.Data = block.VmContext.GetStorageHash().Bytes()
			vm.updateBlock(block, nil, 0)
			quotaLeft = quotaTotal - util.CalcQuotaUsed(quotaTotal, 0, c.quotaLeft, c.quotaRefund, nil)
			err = vm.doSendBlockList(quotaLeft, 0)
			for i, _ := range vm.blockList[1:] {
				vm.blockList[i+1].AccountBlock.Quota = 0
			}
			if err == nil {
				return vm.blockList, NoRetry, nil
			}
		}
	}
	vm.revert(block)
	return nil, NoRetry, err
}

func (vm *VM) sendCall(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	//defer monitor.LogTime("vm", "SendCall", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "sendCall")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok, err := GetPrecompiledContract(block.AccountBlock.ToAddress, block.AccountBlock.Data); ok {
		if err != nil {
			return nil, err
		}
		block.AccountBlock.Fee, err = p.GetFee(block.VmContext, block.AccountBlock)
		if err != nil {
			return nil, err
		}
		if !nodeConfig.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
			return nil, util.ErrInsufficientBalance
		}
		quotaLeft, err = p.DoSend(block.VmContext, block.AccountBlock, quotaLeft)
		if err != nil {
			return nil, err
		}
		block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
		block.VmContext.SubBalance(&ledger.ViteTokenId, block.AccountBlock.Fee)
	} else {
		block.AccountBlock.Fee = helper.Big0
		cost, err := util.IntrinsicGasCost(block.AccountBlock.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
		if !nodeConfig.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
			return nil, util.ErrInsufficientBalance
		}
		block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	}
	quotaUsed := util.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, 0, nil)
	vm.updateBlock(block, nil, quotaUsed)
	return block, nil
}

var (
	ResultSuccess  = byte(0)
	ResultFail     = byte(1)
	ResultDepthErr = byte(2)
)

func getReceiveCallData(db vmctxt_interface.VmDatabase, err error) []byte {
	if err == nil {
		return append(db.GetStorageHash().Bytes(), ResultSuccess)
	} else if err == util.ErrDepth {
		return append(db.GetStorageHash().Bytes(), ResultDepthErr)
	} else {
		return append(db.GetStorageHash().Bytes(), ResultFail)
	}
}

func (vm *VM) receiveCall(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {

	//defer monitor.LogTime("vm", "ReceiveCall", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "receiveCall")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	if checkDepth(block.VmContext, sendBlock) {
		vm.blockList = []*vm_context.VmAccountBlock{block}
		block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		block.AccountBlock.Data = getReceiveCallData(block.VmContext, util.ErrDepth)
		vm.updateBlock(block, util.ErrDepth, 0)
		return vm.blockList, NoRetry, util.ErrDepth
	}
	if p, ok, _ := GetPrecompiledContract(block.AccountBlock.AccountAddress, sendBlock.Data); ok {
		vm.blockList = []*vm_context.VmAccountBlock{block}
		block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		blockListToSend, err := p.DoReceive(block.VmContext, block.AccountBlock, sendBlock)
		if err == nil {
			block.AccountBlock.Data = getReceiveCallData(block.VmContext, err)
			vm.updateBlock(block, err, 0)
			for _, blockToSend := range blockListToSend {
				vm.VmContext.AppendBlock(
					&vm_context.VmAccountBlock{
						util.MakeSendBlock(
							blockToSend.Block,
							blockToSend.ToAddress,
							blockToSend.BlockType,
							blockToSend.Amount,
							blockToSend.TokenId,
							vm.VmContext.GetNewBlockHeight(block),
							blockToSend.Data),
						nil})
			}
			if err = vm.doSendBlockList(0, util.PrecompiledContractsSendGas); err == nil {
				return vm.blockList, NoRetry, nil
			}
		}
		vm.revert(block)
		refundFlag := false
		refundFlag = doRefund(vm, block, sendBlock, p.GetRefundData(), ledger.BlockTypeSendCall)
		block.AccountBlock.Data = getReceiveCallData(block.VmContext, err)
		vm.updateBlock(block, err, 0)
		if refundFlag {
			if refundErr := vm.doSendBlockList(0, util.PrecompiledContractsSendGas); refundErr == nil {
				return vm.blockList, NoRetry, err
			} else {
				monitor.LogEvent("vm", "impossibleReceiveError")
				nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
				return nil, Retry, err
			}
		}
		return vm.blockList, NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			block.VmContext,
			block.AccountBlock.AccountAddress,
			abi.GetPledgeBeneficialAmount(block.VmContext, block.AccountBlock.AccountAddress),
			block.AccountBlock.Difficulty)
		if err != nil {
			return nil, NoRetry, err
		}
		quotaLeft := quotaTotal
		quotaRefund := uint64(0)
		cost, err := util.IntrinsicGasCost(nil, false)
		if err != nil {
			return nil, NoRetry, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, Retry, err
		}
		vm.blockList = []*vm_context.VmAccountBlock{block}
		// add balance, create account if not exist
		block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		// do transfer transaction if account code size is zero
		_, code := util.GetContractCode(block.VmContext, &block.AccountBlock.AccountAddress)
		if len(code) == 0 {
			vm.updateBlock(block, nil, util.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
			return vm.blockList, NoRetry, nil
		}
		// run code
		c := newContract(block, sendBlock, sendBlock.Data, quotaLeft, quotaRefund)
		c.setCallCode(block.AccountBlock.AccountAddress, code)
		_, err = c.run(vm)
		if err == nil {
			block.AccountBlock.Data = getReceiveCallData(block.VmContext, err)
			vm.updateBlock(block, nil, util.CalcQuotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil))
			err = vm.doSendBlockList(quotaTotal-quotaAddition-block.AccountBlock.Quota, 0)
			if err == nil {
				return vm.blockList, NoRetry, nil
			}
		}

		vm.revert(block)

		if err == util.ErrOutOfQuota {
			// if ErrOutOfQuota 3 times, refund with no quota
			block.AccountBlock.Data = getReceiveCallData(block.VmContext, err)
			if receiveBlockHeights, _ := block.VmContext.GetReceiveBlockHeights(&sendBlock.Hash); len(receiveBlockHeights) >= outOfQuotaRetryTime {
				refundFlag := doRefund(vm, block, sendBlock, []byte{}, ledger.BlockTypeSendRefund)
				vm.updateBlock(block, nil, util.CalcQuotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err))
				if refundFlag {
					if refundErr := vm.doSendBlockList(0, util.RefundGas); refundErr == nil {
						return vm.blockList, NoRetry, err
					} else {
						monitor.LogEvent("vm", "impossibleReceiveError")
						nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
						return nil, Retry, err
					}
				}
				return vm.blockList, NoRetry, err
			} else {
				vm.updateBlock(block, err, util.CalcQuotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err))
				return vm.blockList, Retry, err
			}
		}

		refundFlag := doRefund(vm, block, sendBlock, []byte{}, ledger.BlockTypeSendRefund)
		block.AccountBlock.Data = getReceiveCallData(block.VmContext, err)
		vm.updateBlock(block, err, util.CalcQuotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err))
		if refundFlag {
			if refundErr := vm.doSendBlockList(0, util.RefundGas); refundErr == nil {
				return vm.blockList, NoRetry, err
			} else {
				monitor.LogEvent("vm", "impossibleReceiveError")
				nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
				return nil, Retry, err
			}
		}
		return vm.blockList, NoRetry, err
	}
}

func doRefund(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock, refundData []byte, refundBlockType byte) bool {
	refundFlag := false
	if sendBlock.Amount.Sign() > 0 && sendBlock.Fee.Sign() > 0 && sendBlock.TokenId == ledger.ViteTokenId {
		refundAmount := new(big.Int).Add(sendBlock.Amount, sendBlock.Fee)
		vm.VmContext.AppendBlock(
			&vm_context.VmAccountBlock{
				util.MakeSendBlock(
					block.AccountBlock,
					sendBlock.AccountAddress,
					refundBlockType,
					refundAmount,
					ledger.ViteTokenId,
					vm.VmContext.GetNewBlockHeight(block),
					refundData),
				nil})
		block.VmContext.AddBalance(&ledger.ViteTokenId, refundAmount)
		refundFlag = true
	} else {
		if sendBlock.Amount.Sign() > 0 {
			vm.VmContext.AppendBlock(
				&vm_context.VmAccountBlock{
					util.MakeSendBlock(
						block.AccountBlock,
						sendBlock.AccountAddress,
						refundBlockType,
						new(big.Int).Set(sendBlock.Amount),
						sendBlock.TokenId,
						vm.VmContext.GetNewBlockHeight(block),
						refundData),
					nil})
			block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
			refundFlag = true
		}
		if sendBlock.Fee.Sign() > 0 {
			vm.VmContext.AppendBlock(
				&vm_context.VmAccountBlock{
					util.MakeSendBlock(
						block.AccountBlock,
						sendBlock.AccountAddress,
						refundBlockType,
						new(big.Int).Set(sendBlock.Fee),
						ledger.ViteTokenId,
						vm.VmContext.GetNewBlockHeight(block),
						refundData),
					nil})
			block.VmContext.AddBalance(&ledger.ViteTokenId, sendBlock.Fee)
			refundFlag = true
		}
	}
	return refundFlag
}

func (vm *VM) sendReward(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	//defer monitor.LogTime("vm", "SendReward", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "sendReward")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	// check can make transaction
	quotaLeft := quotaTotal
	cost, err := util.IntrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if block.AccountBlock.AccountAddress != types.AddressRegister &&
		block.AccountBlock.AccountAddress != types.AddressMintage {
		return nil, errors.New("invalid account address")
	}
	vm.updateBlock(block, nil, 0)
	return block, nil
}

func (vm *VM) sendRefund(block *vm_context.VmAccountBlock, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	//defer monitor.LogTime("vm", "sendRefund", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "sendRefund")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	block.AccountBlock.Fee = helper.Big0
	cost, err := util.IntrinsicGasCost(block.AccountBlock.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft := quotaTotal
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if !nodeConfig.canTransfer(block.VmContext, block.AccountBlock.AccountAddress, block.AccountBlock.TokenId, block.AccountBlock.Amount, block.AccountBlock.Fee) {
		return nil, util.ErrInsufficientBalance
	}
	block.VmContext.SubBalance(&block.AccountBlock.TokenId, block.AccountBlock.Amount)
	quotaUsed := util.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, 0, nil)
	vm.updateBlock(block, nil, quotaUsed)
	return block, nil
}

func (vm *VM) receiveRefund(block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) (blockList []*vm_context.VmAccountBlock, isRetry bool, err error) {
	//defer monitor.LogTime("vm", "receiveRefund", time.Now())
	var monitorTags []string
	monitorTags = append(monitorTags, "vm", "receiveRefund")
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	// check can make transaction
	quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
		block.VmContext,
		block.AccountBlock.AccountAddress,
		abi.GetPledgeBeneficialAmount(block.VmContext, block.AccountBlock.AccountAddress),
		block.AccountBlock.Difficulty)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := util.IntrinsicGasCost(nil, false)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, Retry, err
	}
	vm.blockList = []*vm_context.VmAccountBlock{block}
	block.VmContext.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
	vm.updateBlock(block, nil, util.CalcQuotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	return vm.blockList, NoRetry, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	_, code := util.GetContractCode(c.block.VmContext, &contractAddr)
	if len(code) > 0 {
		cNew := newContract(c.block, c.sendBlock, c.data, c.quotaLeft, c.quotaRefund)
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
		if err == util.ErrOutOfQuota {
			block.AccountBlock.BlockType = ledger.BlockTypeReceiveError
		} else {
			block.AccountBlock.BlockType = ledger.BlockTypeReceive
		}
	}
}

func (vm *VM) doSendBlockList(quotaLeft uint64, quotaAdditionForOneTx uint64) (err error) {
	db := vm.blockList[0].VmContext
	for i, block := range vm.blockList[1:] {
		db = db.CopyAndFreeze()
		block.VmContext = db
		quotaTotal := quotaLeft + quotaAdditionForOneTx
		switch block.AccountBlock.BlockType {
		case ledger.BlockTypeSendCall:
			vm.blockList[i+1], err = vm.sendCall(block, quotaTotal, quotaAdditionForOneTx)
			if err != nil {
				return err
			}
		case ledger.BlockTypeSendReward:
			vm.blockList[i+1], err = vm.sendReward(block, quotaTotal, quotaAdditionForOneTx)
			if err != nil {
				return err
			}
		case ledger.BlockTypeSendRefund:
			vm.blockList[i+1], err = vm.sendRefund(block, quotaTotal, quotaAdditionForOneTx)
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

func (context *VmContext) AppendBlock(block *vm_context.VmAccountBlock) {
	context.blockList = append(context.blockList, block)
}

func (context *VmContext) GetNewBlockHeight(block *vm_context.VmAccountBlock) uint64 {
	return block.AccountBlock.Height + uint64(len(context.blockList))
}

func calcContractFee(data []byte) (*big.Int, error) {
	return createContractFee, nil
}

func checkDepth(db vmctxt_interface.VmDatabase, sendBlock *ledger.AccountBlock) bool {
	prevBlock := sendBlock
	depth := uint64(1)
	for depth < callDepth {
		if prevBlock == nil {
			panic("cannot find prev block by hash while check depth")
		}
		if util.IsUserAccount(db, prevBlock.AccountAddress) {
			return false
		}
		depth = depth + 1
		prevReceiveBlock := findPrevReceiveBlock(db, prevBlock)
		prevBlock = db.GetAccountBlockByHash(&prevReceiveBlock.FromBlockHash)
		if prevBlock == nil && prevReceiveBlock.Height == 1 && types.IsPrecompiledContractAddress(prevReceiveBlock.AccountAddress) {
			// some precompiled contracts' genesis block does not have prevblock
			return false
		}
	}
	return true
}

func findPrevReceiveBlock(db vmctxt_interface.VmDatabase, sendBlock *ledger.AccountBlock) *ledger.AccountBlock {
	if sendBlock.Height == 1 {
		return nil
	}
	prevHash := sendBlock.PrevHash
	for {
		prevBlock := db.GetAccountBlockByHash(&prevHash)
		if prevBlock == nil {
			panic("cannot find prev block by hash while check depth")
		}
		if prevBlock.IsReceiveBlock() {
			return prevBlock
		}
		prevHash = prevBlock.PrevHash
	}
}
