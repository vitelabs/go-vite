/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common"
	"runtime/debug"

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
				return quota.CalcQuotaForBlock(db, addr, pledgeAmount, difficulty)
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
	sendBlockList []*ledger.AccountBlock
}

type VM struct {
	VMConfig
	abort int32
	VmContext
	i            *Interpreter
	globalStatus *GlobalStatus
}

func NewVM() *VM {
	return &VM{}
}

func printDebugBlockInfo(block *ledger.AccountBlock, result *vm_context.VmAccountBlock, err error) {
	responseBlockList := make([]string, 0)
	if result != nil {
		if result.AccountBlock.IsSendBlock() {
			responseBlockList = append(responseBlockList,
				"{SelfAddr: "+result.AccountBlock.AccountAddress.String()+", "+
					"ToAddr: "+result.AccountBlock.ToAddress.String()+", "+
					"BlockType: "+strconv.FormatInt(int64(result.AccountBlock.BlockType), 10)+", "+
					"Quota: "+strconv.FormatUint(result.AccountBlock.Quota, 10)+", "+
					"Amount: "+result.AccountBlock.Amount.String()+", "+
					"TokenId: "+result.AccountBlock.TokenId.String()+", "+
					"Height: "+strconv.FormatUint(result.AccountBlock.Height, 10)+", "+
					"Data: "+hex.EncodeToString(result.AccountBlock.Data)+", "+
					"Fee: "+result.AccountBlock.Fee.String()+"}")
		} else {
			// TODO add sendBlockList
			responseBlockList = append(responseBlockList,
				"{SelfAddr: "+result.AccountBlock.AccountAddress.String()+", "+
					"FromHash: "+result.AccountBlock.FromBlockHash.String()+", "+
					"BlockType: "+strconv.FormatInt(int64(result.AccountBlock.BlockType), 10)+", "+
					"Quota: "+strconv.FormatUint(result.AccountBlock.Quota, 10)+", "+
					"Height: "+strconv.FormatUint(result.AccountBlock.Height, 10)+", "+
					"Data: "+hex.EncodeToString(result.AccountBlock.Data)+"}")
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

type GlobalStatus struct {
	Seed          interface{}
	SnapshotBlock ledger.SnapshotBlock
}

func (vm *VM) Run(database vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (result []*vm_context.VmAccountBlock, isRetry bool, err error) {
	return nil, false, nil
}

// TODO
func (vm *VM) RunV2(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, status *GlobalStatus) (vmAccountBlock *vm_context.VmAccountBlock, isRetry bool, err error) {
	defer monitor.LogTimerConsuming([]string{"vm", "run"}, time.Now())
	defer func() {
		if nodeConfig.IsDebug {
			printDebugBlockInfo(block, vmAccountBlock, err)
		}
	}()
	if nodeConfig.IsDebug {
		nodeConfig.log.Info("vm run start",
			"blockType", block.BlockType,
			"address", block.AccountAddress.String(),
			"height", block.Height, ""+
				"fromHash", block.FromBlockHash.String())
	}
	blockcopy := block.Copy()
	vm.i = NewInterpreter(db.CurrentSnapshotBlock().Height, false)
	switch block.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		blockcopy.Data = nil
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(db, blockcopy, sendBlock, quota.CalcCreateQuota(sendBlock.Fee))
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(db, blockcopy, sendBlock)
		} else if sendBlock.BlockType == ledger.BlockTypeSendRefund {
			return vm.receiveRefund(db, blockcopy, sendBlock)
		}
	case ledger.BlockTypeSendCreate:
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			db,
			block.AccountAddress,
			abi.GetPledgeBeneficialAmount(db, block.AccountAddress),
			block.Difficulty)
		if err != nil {
			return nil, NoRetry, err
		}
		vmAccountBlock, err = vm.sendCreate(db, blockcopy, true, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return vmAccountBlock, NoRetry, nil
		}
	case ledger.BlockTypeSendCall:
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			db,
			block.AccountAddress,
			abi.GetPledgeBeneficialAmount(db, block.AccountAddress),
			block.Difficulty)
		if err != nil {
			return nil, NoRetry, err
		}
		vmAccountBlock, err = vm.sendCall(db, blockcopy, true, quotaTotal, quotaAddition)
		if err != nil {
			return nil, NoRetry, err
		} else {
			return vmAccountBlock, NoRetry, nil
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
func (vm *VM) sendCreate(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendCreate"}, time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	if useQuota {
		cost, err := util.IntrinsicGasCost(block.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
	}
	var err error
	block.Fee, err = calcContractFee(block.Data)
	if err != nil {
		return nil, err
	}

	gid := util.GetGidFromCreateContractData(block.Data)
	if gid == types.SNAPSHOT_GID {
		return nil, errors.New("invalid consensus group")
	}

	contractType := util.GetContractTypeFromCreateContractData(block.Data)
	if !util.IsExistContractType(contractType) {
		return nil, errors.New("invalid contract type")
	}

	confirmTime := util.GetConfirmTimeFromCreateContractData(block.Data)
	if confirmTime < confirmTimeMin || confirmTime > confirmTimeMax {
		return nil, util.ErrInvalidConfirmTime
	}

	if ContainsStatusCode(util.GetCodeFromCreateContractData(block.Data)) && confirmTime <= 0 {
		return nil, util.ErrInvalidConfirmTime
	}

	if !nodeConfig.canTransfer(db, block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
		return nil, util.ErrInsufficientBalance
	}

	contractAddr := util.NewContractAddress(
		block.AccountAddress,
		block.Height,
		block.PrevHash,
		block.SnapshotHash)

	block.ToAddress = contractAddr
	// sub balance and service fee
	db.SubBalance(&block.TokenId, block.Amount)
	db.SubBalance(&ledger.ViteTokenId, block.Fee)
	vm.updateBlock(db, block, nil, util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	// TODO set contract confirm time
	db.SetContractGid(&gid, &contractAddr)
	return &vm_context.VmAccountBlock{block, db}, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, quotaTotal uint64) (*vm_context.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveCreate"}, time.Now())

	quotaLeft := quotaTotal
	// TODO check new contract address not exists in global status
	if db.IsAddressExisted(&block.AccountAddress) {
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

	gid := util.GetGidFromCreateContractData(sendBlock.Data)
	// TODO get storage of global status
	if !contracts.IsExistGid(db, gid) {
		return nil, NoRetry, errors.New("consensus group not exist")
	}

	// create contract account and add balance
	db.AddBalance(&sendBlock.TokenId, sendBlock.Amount)

	// init contract state and set contract code
	initCode := util.GetCodeFromCreateContractData(sendBlock.Data)
	c := newContract(block, db, sendBlock, initCode, quotaLeft, 0)
	c.setCallCode(block.AccountAddress, initCode)
	code, err := c.run(vm)
	if err == nil && len(code) <= MaxCodeSize {
		code := util.PackContractCode(util.GetContractTypeFromCreateContractData(sendBlock.Data), code)
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = util.UseQuota(c.quotaLeft, codeCost)
		if err == nil {
			db.SetContractCode(code)
			block.Data = db.GetStorageHash().Bytes()
			vm.updateBlock(db, block, nil, 0)
			quotaLeft = quotaTotal - util.CalcQuotaUsed(true, quotaTotal, 0, c.quotaLeft, c.quotaRefund, nil)
			db, err = vm.doSendBlockList(db)
			for i, _ := range vm.sendBlockList {
				vm.sendBlockList[i].Quota = 0
			}
			if err == nil {
				return mergeReceiveBlock(db, block, vm.sendBlockList), NoRetry, nil
			}
		}
	}
	vm.revert(db)
	return nil, NoRetry, err
}

func mergeReceiveBlock(db vmctxt_interface.VmDatabase, receiveBlock *ledger.AccountBlock, sendBlockList []*ledger.AccountBlock) *vm_context.VmAccountBlock {
	if len(sendBlockList) > 0 {
		// TODO merge send block list into receive block
	}
	return &vm_context.VmAccountBlock{receiveBlock, db}
}

func (vm *VM) sendCall(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendCall"}, time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok, err := contracts.GetBuiltinContract(block.ToAddress, block.Data); ok {
		if err != nil {
			return nil, err
		}
		block.Fee, err = p.GetFee(db, block)
		if err != nil {
			return nil, err
		}
		if !nodeConfig.canTransfer(db, block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
			return nil, util.ErrInsufficientBalance
		}
		if useQuota {
			cost, err := p.GetSendQuota(block.Data)
			if err != nil {
				return nil, err
			}
			quotaLeft, err = util.UseQuota(quotaLeft, cost)
			if err != nil {
				return nil, err
			}
		}
		err = p.DoSend(db, block)
		if err != nil {
			return nil, err
		}
		db.SubBalance(&block.TokenId, block.Amount)
		db.SubBalance(&ledger.ViteTokenId, block.Fee)
	} else {
		block.Fee = helper.Big0
		if useQuota {
			cost, err := util.IntrinsicGasCost(block.Data, false)
			if err != nil {
				return nil, err
			}
			quotaLeft, err = util.UseQuota(quotaLeft, cost)
			if err != nil {
				return nil, err
			}
		}
		if !nodeConfig.canTransfer(db, block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
			return nil, util.ErrInsufficientBalance
		}
		db.SubBalance(&block.TokenId, block.Amount)
	}
	quotaUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, 0, nil)
	vm.updateBlock(db, block, nil, quotaUsed)
	return &vm_context.VmAccountBlock{block, db}, nil
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

func (vm *VM) receiveCall(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (*vm_context.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveCall"}, time.Now())

	if checkDepth(db, sendBlock) {
		db.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		block.Data = getReceiveCallData(db, util.ErrDepth)
		vm.updateBlock(db, block, util.ErrDepth, 0)
		return &vm_context.VmAccountBlock{block, db}, NoRetry, util.ErrDepth
	}
	if p, ok, _ := contracts.GetBuiltinContract(block.AccountAddress, sendBlock.Data); ok {
		db.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		blockListToSend, err := p.DoReceive(db, block, sendBlock)
		if err == nil {
			block.Data = getReceiveCallData(db, err)
			vm.updateBlock(db, block, err, 0)
			for _, blockToSend := range blockListToSend {
				vm.VmContext.AppendBlock(
					util.MakeSendBlock(
						block.AccountAddress,
						blockToSend.ToAddress,
						blockToSend.BlockType,
						blockToSend.Amount,
						blockToSend.TokenId,
						blockToSend.Data))
			}
			if db, err = vm.doSendBlockList(db); err == nil {
				return mergeReceiveBlock(db, block, vm.sendBlockList), NoRetry, nil
			}
		}
		vm.revert(db)
		refundFlag := false
		refundFlag = doRefund(vm, db, block, sendBlock, p.GetRefundData(), ledger.BlockTypeSendCall)
		block.Data = getReceiveCallData(db, err)
		vm.updateBlock(db, block, err, 0)
		if refundFlag {
			var refundErr error
			if db, refundErr = vm.doSendBlockList(db); refundErr == nil {
				return mergeReceiveBlock(db, block, vm.sendBlockList), NoRetry, err
			} else {
				monitor.LogEvent("vm", "impossibleReceiveError")
				nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
				return nil, Retry, err
			}
		}
		return &vm_context.VmAccountBlock{block, db}, NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			db,
			block.AccountAddress,
			abi.GetPledgeBeneficialAmount(db, block.AccountAddress),
			block.Difficulty)
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
		// add balance, create account if not exist
		db.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
		// TODO check account type by isContractType instead of getCode
		// do transfer transaction if account code size is zero
		isContract := false
		if !isContract {
			vm.updateBlock(db, block, nil, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
			return &vm_context.VmAccountBlock{block, db}, NoRetry, nil
		}
		// run code
		_, code := util.GetContractCode(db, &block.AccountAddress)
		c := newContract(block, db, sendBlock, sendBlock.Data, quotaLeft, quotaRefund)
		c.setCallCode(block.AccountAddress, code)
		_, err = c.run(vm)
		if err == nil {
			block.Data = getReceiveCallData(db, err)
			vm.updateBlock(db, block, nil, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil))
			db, err = vm.doSendBlockList(db)
			if err == nil {
				return mergeReceiveBlock(db, block, vm.sendBlockList), NoRetry, nil
			}
		}

		vm.revert(db)

		if err == util.ErrOutOfQuota {
			// TODO if is prevBlock is confirmed, block.VmContext.IsConfirmed(block.VmContext.PrevAccountBlock())
			var isPrevBlockConfirmed = true
			// prev account block cannot be nil because this is a contract account, there must be a receive create block at least
			if db.PrevAccountBlock() != nil && !isPrevBlockConfirmed {
				// Contract receive out of quota, current block is not first unconfirmed block, retry next snapshotBlock
				return nil, Retry, err
			} else {
				// Contract receive out of quota, current block is first unconfirmed block, refund with no quota
				block.Data = getReceiveCallData(db, err)
				refundFlag := doRefund(vm, db, block, sendBlock, []byte{}, ledger.BlockTypeSendRefund)
				vm.updateBlock(db, block, nil, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err))
				if refundFlag {
					var refundErr error
					if db, refundErr = vm.doSendBlockList(db); refundErr == nil {
						return mergeReceiveBlock(db, block, vm.sendBlockList), NoRetry, err
					} else {
						monitor.LogEvent("vm", "impossibleReceiveError")
						nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
						return nil, Retry, err
					}
				}
				return &vm_context.VmAccountBlock{block, db}, NoRetry, err
			}
		}

		refundFlag := doRefund(vm, db, block, sendBlock, []byte{}, ledger.BlockTypeSendRefund)
		block.Data = getReceiveCallData(db, err)
		vm.updateBlock(db, block, err, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err))
		if refundFlag {
			var refundErr error
			if db, refundErr = vm.doSendBlockList(db); refundErr == nil {
				return mergeReceiveBlock(db, block, vm.sendBlockList), NoRetry, err
			} else {
				monitor.LogEvent("vm", "impossibleReceiveError")
				nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
				return nil, Retry, err
			}
		}
		return &vm_context.VmAccountBlock{block, db}, NoRetry, err
	}
}

func doRefund(vm *VM, db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, refundData []byte, refundBlockType byte) bool {
	refundFlag := false
	if sendBlock.Amount.Sign() > 0 && sendBlock.Fee.Sign() > 0 && sendBlock.TokenId == ledger.ViteTokenId {
		refundAmount := new(big.Int).Add(sendBlock.Amount, sendBlock.Fee)
		vm.VmContext.AppendBlock(
			util.MakeSendBlock(
				block.AccountAddress,
				sendBlock.AccountAddress,
				refundBlockType,
				refundAmount,
				ledger.ViteTokenId,
				refundData))
		db.AddBalance(&ledger.ViteTokenId, refundAmount)
		refundFlag = true
	} else {
		if sendBlock.Amount.Sign() > 0 {
			vm.VmContext.AppendBlock(
				util.MakeSendBlock(
					block.AccountAddress,
					sendBlock.AccountAddress,
					refundBlockType,
					new(big.Int).Set(sendBlock.Amount),
					sendBlock.TokenId,
					refundData))
			db.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
			refundFlag = true
		}
		if sendBlock.Fee.Sign() > 0 {
			vm.VmContext.AppendBlock(
				util.MakeSendBlock(
					block.AccountAddress,
					sendBlock.AccountAddress,
					refundBlockType,
					new(big.Int).Set(sendBlock.Fee),
					ledger.ViteTokenId,
					refundData))
			db.AddBalance(&ledger.ViteTokenId, sendBlock.Fee)
			refundFlag = true
		}
	}
	return refundFlag
}

func (vm *VM) sendReward(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendReward"}, time.Now())

	// check can make transaction
	quotaLeft := quotaTotal
	if useQuota {
		cost, err := util.IntrinsicGasCost(block.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
	}
	if block.AccountAddress != types.AddressConsensusGroup &&
		block.AccountAddress != types.AddressMintage {
		return nil, errors.New("invalid account address")
	}
	vm.updateBlock(db, block, nil, util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, 0, nil))
	return &vm_context.VmAccountBlock{block, db}, nil
}

func (vm *VM) sendRefund(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_context.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendRefund"}, time.Now())
	block.Fee = helper.Big0
	quotaLeft := quotaTotal
	if useQuota {
		cost, err := util.IntrinsicGasCost(block.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
	}
	if !nodeConfig.canTransfer(db, block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
		return nil, util.ErrInsufficientBalance
	}
	db.SubBalance(&block.TokenId, block.Amount)
	quotaUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, 0, nil)
	vm.updateBlock(db, block, nil, quotaUsed)
	return &vm_context.VmAccountBlock{block, db}, nil
}

func (vm *VM) receiveRefund(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (*vm_context.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveRefund"}, time.Now())

	// check can make transaction
	quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
		db,
		block.AccountAddress,
		abi.GetPledgeBeneficialAmount(db, block.AccountAddress),
		block.Difficulty)
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
	db.AddBalance(&sendBlock.TokenId, sendBlock.Amount)
	vm.updateBlock(db, block, nil, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	return &vm_context.VmAccountBlock{block, db}, NoRetry, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	_, code := util.GetContractCode(c.db, &contractAddr)
	if len(code) > 0 {
		cNew := newContract(c.block, c.db, c.sendBlock, c.data, c.quotaLeft, c.quotaRefund)
		cNew.setCallCode(contractAddr, code)
		ret, err = cNew.run(vm)
		c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
		return ret, err
	}
	return nil, nil
}

func (vm *VM) updateBlock(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, err error, quotaUsed uint64) {
	block.Quota = quotaUsed
	if block.IsReceiveBlock() {
		block.StateHash = *db.GetStorageHash()
		block.LogHash = db.GetLogListHash()
		if err == util.ErrOutOfQuota {
			block.BlockType = ledger.BlockTypeReceiveError
		} else {
			block.BlockType = ledger.BlockTypeReceive
		}
	}
}

func (vm *VM) doSendBlockList(db vmctxt_interface.VmDatabase) (newDb vmctxt_interface.VmDatabase, err error) {
	if len(vm.sendBlockList) == 0 {
		return db, nil
	}
	for i, block := range vm.sendBlockList {
		var sendBlock *vm_context.VmAccountBlock
		switch block.BlockType {
		case ledger.BlockTypeSendCall:
			sendBlock, err = vm.sendCall(db, block, false, 0, 0)
			if err != nil {
				return db, err
			}
		case ledger.BlockTypeSendReward:
			sendBlock, err = vm.sendReward(db, block, false, 0, 0)
			if err != nil {
				return db, err
			}
		case ledger.BlockTypeSendRefund:
			sendBlock, err = vm.sendRefund(db, block, false, 0, 0)
			if err != nil {
				return db, err
			}
		}
		vm.sendBlockList[i] = sendBlock.AccountBlock
		db = sendBlock.VmContext
	}
	return db, nil
}

func (vm *VM) revert(db vmctxt_interface.VmDatabase) {
	vm.sendBlockList = nil
	db.Reset()
}

func (context *VmContext) AppendBlock(block *ledger.AccountBlock) {
	context.sendBlockList = append(context.sendBlockList, block)
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
		if prevBlock == nil && prevReceiveBlock.Height == 1 && types.IsBuiltinContractAddrInUse(prevReceiveBlock.AccountAddress) {
			// some built-in contracts' genesis block does not have prevblock
			return false
		}
	}
	return true
}

func findPrevReceiveBlock(db vmctxt_interface.VmDatabase, sendBlock *ledger.AccountBlock) *ledger.AccountBlock {
	// TODO check vmcontext method change
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

func (vm *VM) OffChainReader(db vmctxt_interface.VmDatabase, code []byte, data []byte) (result []byte, err error) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			nodeConfig.log.Error("offchain reader panic",
				"err", err,
				"addr", db.Address(),
				"snapshotHash", db.CurrentSnapshotBlock().Hash,
				"code", hex.EncodeToString(code),
				"data", hex.EncodeToString(data))
			result = nil
			err = errors.New("offchain reader panic")
		}
	}()
	vm.i = NewInterpreter(db.CurrentSnapshotBlock().Height, true)
	c := newContract(&ledger.AccountBlock{AccountAddress: *db.Address()}, db, &ledger.AccountBlock{ToAddress: *db.Address()}, data, offChainReaderGas, 0)
	c.setCallCode(*db.Address(), code)
	return c.run(vm)
}
