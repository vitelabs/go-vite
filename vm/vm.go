/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/vm_db"
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
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
)

type VMConfig struct {
	Debug bool
}

type NodeConfig struct {
	isTest      bool
	calcQuota   func(db vm_db.VmDb, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error)
	canTransfer func(db vm_db.VmDb, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool

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
			calcQuota: func(db vm_db.VmDb, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
				return 1000000, 0, nil
			},
			canTransfer: func(db vm_db.VmDb, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
				return true
			},
		}
	} else {
		nodeConfig = NodeConfig{
			isTest: isTest,
			calcQuota: func(db vm_db.VmDb, pledgeAmount *big.Int, difficulty *big.Int) (quotaTotal uint64, quotaAddition uint64, err error) {
				return quota.CalcQuotaForBlock(db, pledgeAmount, difficulty)
			},
			canTransfer: func(db vm_db.VmDb, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
				if feeAmount.Sign() == 0 {
					b, err := db.GetBalance(&tokenTypeId)
					util.DealWithErr(err)
					return tokenAmount.Cmp(b) <= 0
				}
				if util.IsViteToken(tokenTypeId) {
					balance := new(big.Int).Add(tokenAmount, feeAmount)
					b, err := db.GetBalance(&tokenTypeId)
					util.DealWithErr(err)
					return balance.Cmp(b) <= 0
				}
				amountB, err := db.GetBalance(&tokenTypeId)
				util.DealWithErr(err)
				feeB, err := db.GetBalance(&ledger.ViteTokenId)
				util.DealWithErr(err)
				return tokenAmount.Cmp(amountB) <= 0 && feeAmount.Cmp(feeB) <= 0
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
	globalStatus *util.GlobalStatus
}

func NewVM() *VM {
	return &VM{}
}

func printDebugBlockInfo(block *ledger.AccountBlock, result *vm_db.VmAccountBlock, err error) {
	var str string
	if result != nil {
		if result.AccountBlock.IsSendBlock() {
			str = "{SelfAddr: " + result.AccountBlock.AccountAddress.String() + ", " +
				"ToAddr: " + result.AccountBlock.ToAddress.String() + ", " +
				"BlockType: " + strconv.FormatInt(int64(result.AccountBlock.BlockType), 10) + ", " +
				"Quota: " + strconv.FormatUint(result.AccountBlock.Quota, 10) + ", " +
				"Amount: " + result.AccountBlock.Amount.String() + ", " +
				"TokenId: " + result.AccountBlock.TokenId.String() + ", " +
				"Height: " + strconv.FormatUint(result.AccountBlock.Height, 10) + ", " +
				"Data: " + hex.EncodeToString(result.AccountBlock.Data) + ", " +
				"Fee: " + result.AccountBlock.Fee.String() + "}"
		} else {
			if len(result.AccountBlock.SendBlockList) > 0 {
				str = "["
				for _, sendBlock := range result.AccountBlock.SendBlockList {
					str = str + "{ToAddr:" + sendBlock.ToAddress.String() + ", " +
						"BlockType:" + strconv.FormatInt(int64(sendBlock.BlockType), 10) + ", " +
						"Data:" + hex.EncodeToString(sendBlock.Data) + ", " +
						"Amount:" + sendBlock.Amount.String() + ", " +
						"TokenId:" + sendBlock.TokenId.String() + ", " +
						"Fee:" + sendBlock.Fee.String() + "}"
				}
				str = str + "]"
			}
			str = "{SelfAddr: " + result.AccountBlock.AccountAddress.String() + ", " +
				"FromHash: " + result.AccountBlock.FromBlockHash.String() + ", " +
				"BlockType: " + strconv.FormatInt(int64(result.AccountBlock.BlockType), 10) + ", " +
				"Quota: " + strconv.FormatUint(result.AccountBlock.Quota, 10) + ", " +
				"Height: " + strconv.FormatUint(result.AccountBlock.Height, 10) + ", " +
				"Data: " + hex.EncodeToString(result.AccountBlock.Data) + ", " +
				"SendBlockList: " + str + "}"
		}
	}
	nodeConfig.log.Info("vm run stop",
		"blockType", block.BlockType,
		"address", block.AccountAddress.String(),
		"height", block.Height,
		"fromHash", block.FromBlockHash.String(),
		"err", err,
		"block", str,
	)
}

func getContractMeta(db vm_db.VmDb) *ledger.ContractMeta {
	ok, err := db.IsContractAccount()
	util.DealWithErr(err)
	if !ok {
		return nil
	}
	meta, err := db.GetContractMeta()
	util.DealWithErr(err)
	return meta
}

func (vm *VM) RunV2(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, status *util.GlobalStatus) (vmAccountBlock *vm_db.VmAccountBlock, isRetry bool, err error) {
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
	sb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	vm.i = NewInterpreter(sb.Height, false)
	vm.globalStatus = status
	switch block.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		blockcopy.Data = nil
		contractMeta := getContractMeta(db)
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(db, blockcopy, sendBlock, quota.CalcCreateQuota(sendBlock.Fee), contractMeta)
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(db, blockcopy, sendBlock, contractMeta)
		} else if sendBlock.BlockType == ledger.BlockTypeSendRefund {
			return vm.receiveRefund(db, blockcopy, sendBlock, contractMeta)
		}
	case ledger.BlockTypeSendCreate:
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			db,
			getPledgeAmount(db),
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
			getPledgeAmount(db),
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
		return nil, NoRetry, util.ErrTransactionTypeNotSupport
	}
	return nil, NoRetry, util.ErrTransactionTypeNotSupport
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendCreate"}, time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	if useQuota {
		cost, err := gasNormalSendCall(block)
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

	if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
		return nil, util.ErrInsufficientBalance
	}

	contractAddr := util.NewContractAddress(
		block.AccountAddress,
		block.Height,
		block.PrevHash)

	block.ToAddress = contractAddr
	// sub balance and service fee
	util.SubBalance(db, &block.TokenId, block.Amount)
	util.SubBalance(db, &ledger.ViteTokenId, block.Fee)
	vm.updateBlock(db, block, nil, util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	db.SetContractMeta(contractAddr, &ledger.ContractMeta{&gid, confirmTime})
	return &vm_db.VmAccountBlock{block, db}, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, quotaTotal uint64, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveCreate"}, time.Now())

	quotaLeft := quotaTotal
	prev, err := db.PrevAccountBlock()
	util.DealWithErr(err)
	if prev != nil {
		return nil, NoRetry, util.ErrAddressCollision
	}
	// check can make transaction
	cost, err := gasReceiveCreate(block, meta)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, NoRetry, err
	}

	// create contract account and add balance
	util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)

	// init contract state_bak and set contract code
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
			block.Data = db.GetReceiptHash().Bytes()
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

func mergeReceiveBlock(db vm_db.VmDb, receiveBlock *ledger.AccountBlock, sendBlockList []*ledger.AccountBlock) *vm_db.VmAccountBlock {
	receiveBlock.SendBlockList = sendBlockList
	return &vm_db.VmAccountBlock{receiveBlock, db}
}

func (vm *VM) sendCall(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendCall"}, time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok, err := contracts.GetBuiltinContract(block.ToAddress, block.Data); ok {
		if err != nil {
			return nil, err
		}
		block.Fee, err = p.GetFee(block)
		if err != nil {
			return nil, err
		}
		if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
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
		util.SubBalance(db, &block.TokenId, block.Amount)
		util.SubBalance(db, &ledger.ViteTokenId, block.Fee)
	} else {
		block.Fee = helper.Big0
		if useQuota {
			cost, err := gasNormalSendCall(block)
			if err != nil {
				return nil, err
			}
			quotaLeft, err = util.UseQuota(quotaLeft, cost)
			if err != nil {
				return nil, err
			}
		}
		if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
			return nil, util.ErrInsufficientBalance
		}
		util.SubBalance(db, &block.TokenId, block.Amount)
	}
	quotaUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, 0, nil)
	vm.updateBlock(db, block, nil, quotaUsed)
	return &vm_db.VmAccountBlock{block, db}, nil
}

var (
	ResultSuccess  = byte(0)
	ResultFail     = byte(1)
	ResultDepthErr = byte(2)
)

func getReceiveCallData(db vm_db.VmDb, err error) []byte {
	if err == nil {
		return append(db.GetReceiptHash().Bytes(), ResultSuccess)
	} else if err == util.ErrDepth {
		return append(db.GetReceiptHash().Bytes(), ResultDepthErr)
	} else {
		return append(db.GetReceiptHash().Bytes(), ResultFail)
	}
}

func (vm *VM) receiveCall(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveCall"}, time.Now())

	if checkDepth(db, sendBlock) {
		util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
		block.Data = getReceiveCallData(db, util.ErrDepth)
		vm.updateBlock(db, block, util.ErrDepth, 0)
		return &vm_db.VmAccountBlock{block, db}, NoRetry, util.ErrDepth
	}
	if p, ok, _ := contracts.GetBuiltinContract(block.AccountAddress, sendBlock.Data); ok {
		util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
		blockListToSend, err := p.DoReceive(db, block, sendBlock, vm.globalStatus)
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
		return &vm_db.VmAccountBlock{block, db}, NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
			db,
			getPledgeAmount(db),
			block.Difficulty)
		if err != nil {
			return nil, NoRetry, err
		}
		quotaLeft := quotaTotal
		quotaRefund := uint64(0)
		cost, err := gasReceive(block, meta)
		if err != nil {
			return nil, NoRetry, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, Retry, err
		}
		// add balance, create account if not exist
		util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
		// do transfer transaction if account code size is zero
		if meta == nil {
			vm.updateBlock(db, block, nil, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
			return &vm_db.VmAccountBlock{block, db}, NoRetry, nil
		}
		// run code
		_, code := util.GetContractCode(db, &block.AccountAddress, nil)
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
			unConfirmedList := db.GetUnconfirmedBlocks()
			if len(unConfirmedList) > 0 {
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
				return &vm_db.VmAccountBlock{block, db}, NoRetry, err
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
		return &vm_db.VmAccountBlock{block, db}, NoRetry, err
	}
}

func doRefund(vm *VM, db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, refundData []byte, refundBlockType byte) bool {
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
		util.AddBalance(db, &ledger.ViteTokenId, refundAmount)
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
			util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
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
			util.AddBalance(db, &ledger.ViteTokenId, sendBlock.Fee)
			refundFlag = true
		}
	}
	return refundFlag
}

func (vm *VM) sendReward(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendReward"}, time.Now())

	// check can make transaction
	quotaLeft := quotaTotal
	if useQuota {
		cost, err := gasNormalSendCall(block)
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
	return &vm_db.VmAccountBlock{block, db}, nil
}

func (vm *VM) sendRefund(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendRefund"}, time.Now())
	block.Fee = helper.Big0
	quotaLeft := quotaTotal
	if useQuota {
		cost, err := gasNormalSendCall(block)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
	}
	if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
		return nil, util.ErrInsufficientBalance
	}
	util.SubBalance(db, &block.TokenId, block.Amount)
	quotaUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, 0, nil)
	vm.updateBlock(db, block, nil, quotaUsed)
	return &vm_db.VmAccountBlock{block, db}, nil
}

func (vm *VM) receiveRefund(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveRefund"}, time.Now())
	// check can make transaction
	quotaTotal, quotaAddition, err := nodeConfig.calcQuota(
		db,
		getPledgeAmount(db),
		block.Difficulty)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := gasReceive(block, meta)
	if err != nil {
		return nil, NoRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, Retry, err
	}
	util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
	vm.updateBlock(db, block, nil, util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil))
	return &vm_db.VmAccountBlock{block, db}, NoRetry, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	_, code := util.GetContractCode(c.db, &contractAddr, vm.globalStatus)
	if len(code) > 0 {
		cNew := newContract(c.block, c.db, c.sendBlock, c.data, c.quotaLeft, c.quotaRefund)
		cNew.setCallCode(contractAddr, code)
		ret, err = cNew.run(vm)
		c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
		return ret, err
	}
	return nil, nil
}

func (vm *VM) updateBlock(db vm_db.VmDb, block *ledger.AccountBlock, err error, quotaUsed uint64) {
	block.Quota = quotaUsed
	if block.IsReceiveBlock() {
		block.LogHash = db.GetLogListHash()
		if err == util.ErrOutOfQuota {
			block.BlockType = ledger.BlockTypeReceiveError
		} else {
			block.BlockType = ledger.BlockTypeReceive
		}
	}
}

func (vm *VM) doSendBlockList(db vm_db.VmDb) (newDb vm_db.VmDb, err error) {
	if len(vm.sendBlockList) == 0 {
		return db, nil
	}
	for i, block := range vm.sendBlockList {
		var sendBlock *vm_db.VmAccountBlock
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
		db = sendBlock.VmDb
	}
	return db, nil
}

func (vm *VM) revert(db vm_db.VmDb) {
	vm.sendBlockList = nil
	db.Reset()
}

func (context *VmContext) AppendBlock(block *ledger.AccountBlock) {
	context.sendBlockList = append(context.sendBlockList, block)
}

func calcContractFee(data []byte) (*big.Int, error) {
	return createContractFee, nil
}

func checkDepth(db vm_db.VmDb, sendBlock *ledger.AccountBlock) bool {
	depth, err := db.GetCallDepth(&sendBlock.Hash)
	util.DealWithErr(err)
	return depth >= callDepth
}

func (vm *VM) OffChainReader(db vm_db.VmDb, code []byte, data []byte) (result []byte, err error) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			nodeConfig.log.Error("offchain reader panic",
				"err", err,
				"addr", db.Address(),
				"code", hex.EncodeToString(code),
				"data", hex.EncodeToString(data))
			result = nil
			err = errors.New("offchain reader panic")
		}
	}()
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	vm.i = NewInterpreter(sb.Height, true)
	c := newContract(&ledger.AccountBlock{AccountAddress: *db.Address()}, db, &ledger.AccountBlock{ToAddress: *db.Address()}, data, offChainReaderGas, 0)
	c.setCallCode(*db.Address(), code)
	return c.run(vm)
}

func getPledgeAmount(db vm_db.VmDb) *big.Int {
	pledgeAmount, err := db.GetPledgeAmount(db.Address())
	util.DealWithErr(err)
	return pledgeAmount
}
