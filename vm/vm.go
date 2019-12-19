// Package vm implements the vite virtual machine
package vm

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/vm/abi"
	"runtime/debug"
	"sync"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/vm_db"

	"math/big"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
)

// NodeConfig holds the global status of vm.
type NodeConfig struct {
	isTest  bool
	IsDebug bool
	// interpreterLog is used to print run log of interpreters under debug mode
	interpreterLog log15.Logger
	log            log15.Logger

	// canTransfer is the signature of a transfer guard function.
	// This method checks whether there are enough funds in current address
	// account to pay for transfer token amount and fee.
	// Note: Fee amount is payed in vite coin.
	canTransfer func(db vm_db.VmDb, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool

	ContractABIMap   map[types.Address]abi.ABIContract
	contractABIMapRW sync.RWMutex
}

// AddContractABI method is used for debug
func AddContractABI(addr types.Address, info abi.ABIContract) {
	nodeConfig.contractABIMapRW.Lock()
	defer nodeConfig.contractABIMapRW.Unlock()
	if nodeConfig.ContractABIMap == nil {
		nodeConfig.ContractABIMap = make(map[types.Address]abi.ABIContract)
	}
	nodeConfig.ContractABIMap[addr] = info
}

// GetContractABI method is used for debug
func GetContractABI(addr types.Address) (abi.ABIContract, bool) {
	nodeConfig.contractABIMapRW.RLock()
	defer nodeConfig.contractABIMapRW.RUnlock()
	contractAbi, ok := nodeConfig.ContractABIMap[addr]
	return contractAbi, ok
}

var nodeConfig NodeConfig

// IsTest returns whether node is currently running under test mode or not.
func IsTest() bool {
	return nodeConfig.isTest
}

// InitVMConfig init global status of vm. This method is supposed be called when
// the node started.
// Parameters:
//   isTest: test mode, quota and balance is not checked under test mode.
//   isTestParam: use test params for built-in contracts.
//   isQuotaTestParam: use test params for quota calculation.
//   isDebug: print debug log.
//   datadir: print debug log under this directory.
func InitVMConfig(isTest bool, isTestParam bool, isQuotaTestParam bool, isDebug bool, dataDir string) {
	if isTest {
		nodeConfig = NodeConfig{
			isTest: isTest,
			canTransfer: func(db vm_db.VmDb, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
				return true
			},
		}
	} else {
		nodeConfig = NodeConfig{
			isTest: isTest,
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
	quota.InitQuotaConfig(isTest, isQuotaTestParam)
	nodeConfig.IsDebug = isDebug
	if isDebug {
		initLog(dataDir, "dbug")
	}
}

func initLog(dir, lvl string) {
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

type vmContext struct {
	sendBlockList []*ledger.AccountBlock
}

// VM holds the runtime information of vite vm and provides the necessary tools
// to run a transfer transaction of a call contract transaction. It also
// provides an offchain getter method to read contract storage without sending
// a transaction.
// Node: The VM instance should never be reused and is not thread safe.
type VM struct {
	abort int32
	// vmContext holds middle results during current execution
	vmContext
	// interpreter instance,varied by current snapshot block height
	i *interpreter
	// globalStatus holds world status during current execution
	globalStatus util.GlobalStatus
	// reader holds a consensus reader instance, used for reward calculation
	reader util.ConsensusReader
	// latest snapshot block height, used for fork check
	latestSnapshotHeight uint64
	gasTable             *util.QuotaTable
}

// NewVM is a constructor of VM. This method is called before running an
// execution.
func NewVM(cr util.ConsensusReader) *VM {
	return &VM{reader: cr}
}

// GlobalStatus is a getter method.
func (vm *VM) GlobalStatus() util.GlobalStatus {
	return vm.globalStatus
}

// ConsensusReader is a getter method.
func (vm *VM) ConsensusReader() util.ConsensusReader {
	return vm.reader
}

// RunV2 method executes an account block, checks parameters,
// performs balance change and storage change, updates specific
// fields of the account block.
// This method is used to both executes an account block and
// verify an account block.
// Parameters:
//   db: current status, including current account address,
//       prev account block and latest snapshot block.
//   block: account block to be executed.
//   sendBlock: when executing a receive block,
//       sendBlock is the referring send block.
//   status: world status, only presents when executing
//       a contract receive block and contract confirm time
//       is greater than zero.
// Returns:
//   vmAccountBlock: execute result, including db and block.
//   isRetry: whether this block should be executed again later.
//   err: specific error occurred during execution.
// Notes:
//   1. Input block and sendBlock will not be changed during execution.
//      The return block in vmAccountBlock is a copy of input block.
//   2. Return value of vmAccountBlock should be inserted into chain
//      regardless of isRetry return value and err return value.
//   3. Block should be executed again later if isRetry is true,
//      regardless of vmAccountBlock return value and err return value.
//   4. This method panics if chain forked during execution, retry later
//      if panics.
func (vm *VM) RunV2(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, status util.GlobalStatus) (vmAccountBlock *vm_db.VmAccountBlock, isRetry bool, err error) {
	defer monitor.LogTimerConsuming([]string{"vm", "run"}, time.Now())
	defer func() {
		db.Finish()
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
	sb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	vm.latestSnapshotHeight = sb.Height
	vm.gasTable = util.QuotaTableByHeight(sb.Height)
	// In case vm will update some fields of block, make a copy of block.
	blockCopy := block.Copy()
	if blockCopy.IsSendBlock() {
		if blockCopy.BlockType == ledger.BlockTypeSendCreate {
			quotaTotal, quotaAddition, err := quota.GetQuotaForBlock(
				db,
				blockCopy.AccountAddress,
				getStakeBeneficialAmount(db),
				blockCopy.Difficulty,
				sb.Height)
			if err != nil {
				return nil, noRetry, err
			}
			vmAccountBlock, err = vm.sendCreate(db, blockCopy, true, quotaTotal, quotaAddition)
			return vmAccountBlock, noRetry, err
		}
		if blockCopy.BlockType == ledger.BlockTypeSendCall {
			quotaTotal, quotaAddition, err := quota.GetQuotaForBlock(
				db,
				blockCopy.AccountAddress,
				getStakeBeneficialAmount(db),
				blockCopy.Difficulty,
				sb.Height)
			if err != nil {
				return nil, noRetry, err
			}
			vmAccountBlock, err = vm.sendCall(db, blockCopy, true, quotaTotal, quotaAddition)
			return vmAccountBlock, noRetry, err
		}
	} else {
		// New interpreter instance according to latest snapshot block height.
		vm.i = newInterpreter(sb.Height, false)
		vm.globalStatus = status
		blockCopy.Data = nil
		contractMeta := getContractMeta(db)
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(db, blockCopy, sendBlock, contractMeta)
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall {
			return vm.receiveCall(db, blockCopy, sendBlock, contractMeta)
		} else if sendBlock.BlockType == ledger.BlockTypeSendReward {
			if !fork.IsSeedFork(sb.Height) {
				return vm.receiveCall(db, blockCopy, sendBlock, contractMeta)
			}
			return vm.receiveReward(db, blockCopy, sendBlock, contractMeta)
		} else if sendBlock.BlockType == ledger.BlockTypeSendRefund {
			return vm.receiveRefund(db, blockCopy, sendBlock, contractMeta)
		}
	}
	// Notice that send reward and send refund type is not supposed to be
	// dealt with in this method.
	return nil, noRetry, util.ErrTransactionTypeNotSupport
}

// Cancel method stops the running contract receive
func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// sendCreate executes a send transaction to create a contract.
// This method whether returns error or returns the success send create block.
// Parameters:
//   db: current status.
//   block: send block to be executed.
//   useQuota: whether this transaction consumes quota. A transaction sent
//     by user consumes quota, while a transaction sent by a receive
//     transaction of a contract does not consume quota since quota is
//     consumed in receive transaction.
//   quotaTotal: total quota this transaction can use during execution.
//     quotaTotal is consists of stake quota and PoW quota.
//   quotaAddition: PoW quota this transaction can use.
func (vm *VM) sendCreate(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendCreate"}, time.Now())
	// Check quota for transaction.
	quotaLeft := quotaTotal
	if useQuota {
		cost, err := gasSendCreate(block, vm.gasTable)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
	}

	// Check params.
	isSeedFork := fork.IsSeedFork(vm.latestSnapshotHeight)
	if !isSeedFork {
		if len(block.Data) < util.CreateContractDataLengthMin {
			return nil, util.ErrInvalidMethodParam
		}
	} else {
		if len(block.Data) < util.CreateContractDataLengthMinRand {
			return nil, util.ErrInvalidMethodParam
		}
	}
	gid := util.GetGidFromCreateContractData(block.Data)
	if gid == types.SNAPSHOT_GID {
		return nil, util.ErrInvalidMethodParam
	}
	contractType := util.GetContractTypeFromCreateContractData(block.Data)
	if !util.IsExistContractType(contractType) {
		return nil, util.ErrInvalidMethodParam
	}
	snapshotCount := util.GetSnapshotCountFromCreateContractData(block.Data)
	if snapshotCount < snapshotCountMin || snapshotCount > snapshotCountMax {
		return nil, util.ErrInvalidResponseLatency
	}
	quotaMultiplier := util.GetQuotaMultiplierFromCreateContractData(block.Data, vm.latestSnapshotHeight)
	if quotaMultiplier < 10 || quotaMultiplier > 100 {
		return nil, util.ErrInvalidQuotaMultiplier
	}

	snapshotWithSeedCount := snapshotCount
	if !isSeedFork {
		if ContainsStatusCode(util.GetCodeFromCreateContractData(block.Data, vm.latestSnapshotHeight)) && snapshotCount <= 0 {
			return nil, util.ErrInvalidResponseLatency
		}
	} else {
		snapshotWithSeedCount = util.GetSnapshotWithSeedCountCountFromCreateContractData(block.Data)
		if snapshotWithSeedCount < snapshotWithSeedCountMin || snapshotWithSeedCount > snapshotWithSeedCountMax || snapshotCount < snapshotWithSeedCount {
			return nil, util.ErrInvalidRandomDegree
		}
		code := util.GetCodeFromCreateContractData(block.Data, vm.latestSnapshotHeight)
		requireSnapshot, requireSnapshotWithSeed := ContainsCertainStatusCode(code)
		if requireSnapshot && snapshotCount <= 0 {
			return nil, util.ErrInvalidResponseLatency
		}
		if requireSnapshotWithSeed && snapshotWithSeedCount <= 0 {
			return nil, util.ErrInvalidRandomDegree
		}
	}

	// Set contract fee.
	var err error
	block.Fee, err = calcContractFee(block.Data)
	if err != nil {
		return nil, err
	}
	// Check balance.
	if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
		return nil, util.ErrInsufficientBalance
	}
	// Generate a new contract address and set block field.
	contractAddr := util.NewContractAddress(
		block.AccountAddress,
		block.Height,
		block.PrevHash)
	block.ToAddress = contractAddr
	// Deduct balance and service fee.
	util.SubBalance(db, &block.TokenId, block.Amount)
	util.SubBalance(db, &ledger.ViteTokenId, block.Fee)

	qStakeUsed, qUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, nil)
	vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
	// Set contract meta at send block, so that contract block producer module
	// will be informed of the binding between gid and the new contract.
	db.SetContractMeta(contractAddr, &ledger.ContractMeta{Gid: gid, SendConfirmedTimes: snapshotCount, QuotaRatio: quotaMultiplier, SeedConfirmedTimes: snapshotWithSeedCount})
	return &vm_db.VmAccountBlock{block, db}, nil
}

// receiveCreate executes a receive transaction to create a contract.
// Parameters:
//   db: current status.
//   block: receive block to be executed.
//   sendBlock: send create block.
//   meta: contract meta set by send create block.
func (vm *VM) receiveCreate(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveCreate"}, time.Now())
	quotaLeft := quota.QuotaForCreateContractResponse
	// Check contract address collision.
	prev, err := db.PrevAccountBlock()
	util.DealWithErr(err)
	if prev != nil {
		return nil, noRetry, util.ErrAddressCollision
	}
	// Check quota for transaction.
	cost, err := gasReceiveCreate(block, meta, vm.gasTable)
	if err != nil {
		return nil, noRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, noRetry, err
	}

	// Create contract account and add balance.
	util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)

	// init contract state_bak and set contract code
	initCode := util.GetCodeFromCreateContractData(sendBlock.Data, vm.latestSnapshotHeight)
	c := newContract(block, db, sendBlock, initCode, quotaLeft)
	c.setCallCode(block.AccountAddress, initCode)
	code, err := c.run(vm)
	if err == nil && len(code) <= maxCodeSize {
		code := util.PackContractCode(util.GetContractTypeFromCreateContractData(sendBlock.Data), code)
		codeCost := uint64(len(code)) * vm.gasTable.CodeQuota
		c.quotaLeft, err = util.UseQuota(c.quotaLeft, codeCost)
		if err == nil {
			db.SetContractCode(code)
			vm.updateBlock(db, block, err, 0, 0)
			db, err = vm.doSendBlockList(db)
			if err == nil {
				block.Data = getReceiveCallData(db, err)
				return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, nil
			}
		}
	}
	if err == nil && len(code) > maxCodeSize && fork.IsEarthFork(vm.latestSnapshotHeight) {
		err = util.ErrInvalidCodeLength
	}
	vm.revert(db)

	// try refund
	vm.updateBlock(db, block, err, 0, 0)
	if sendBlock.Amount.Sign() > 0 {
		vm.vmContext.AppendBlock(
			util.MakeRequestBlock(
				block.AccountAddress,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendRefund,
				new(big.Int).Set(sendBlock.Amount),
				sendBlock.TokenId,
				[]byte{}))
		util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
		var refundErr error
		if db, refundErr = vm.doSendBlockList(db); refundErr == nil {
			block.Data = getReceiveCallData(db, err)
			return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, err
		}
		monitor.LogEvent("vm", "impossibleReceiveError")
		nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
		return nil, retry, err
	}
	block.Data = getReceiveCallData(db, err)
	return &vm_db.VmAccountBlock{block, db}, noRetry, err
}

func mergeReceiveBlock(db vm_db.VmDb, receiveBlock *ledger.AccountBlock, sendBlockList []*ledger.AccountBlock) *vm_db.VmAccountBlock {
	receiveBlock.SendBlockList = sendBlockList
	return &vm_db.VmAccountBlock{receiveBlock, db}
}

func (vm *VM) sendCall(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendCall"}, time.Now())
	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok, err := contracts.GetBuiltinContractMethod(block.ToAddress, block.Data, vm.latestSnapshotHeight); ok {
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
			cost, err := p.GetSendQuota(block.Data, vm.gasTable)
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
			quotaLeft, err = useQuotaForSend(block, db, quotaLeft, vm.gasTable)
			if err != nil {
				return nil, err
			}
		}
		if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
			return nil, util.ErrInsufficientBalance
		}
		util.SubBalance(db, &block.TokenId, block.Amount)
	}
	qStakeUsed, qUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, nil)
	vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
	return &vm_db.VmAccountBlock{block, db}, nil
}

var (
	resultSuccess  = byte(0)
	resultFail     = byte(1)
	resultDepthErr = byte(2)
)

func getReceiveCallData(db vm_db.VmDb, err error) []byte {
	if err == nil {
		return append(db.GetReceiptHash().Bytes(), resultSuccess)
	} else if err == util.ErrDepth {
		return append(db.GetReceiptHash().Bytes(), resultDepthErr)
	} else {
		return append(db.GetReceiptHash().Bytes(), resultFail)
	}
}

func (vm *VM) receiveCall(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveCall"}, time.Now())

	if checkDepth(db, sendBlock) {
		util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
		block.Data = getReceiveCallData(db, util.ErrDepth)
		vm.updateBlock(db, block, util.ErrDepth, 0, 0)
		return &vm_db.VmAccountBlock{block, db}, noRetry, util.ErrDepth
	}
	if p, ok, _ := contracts.GetBuiltinContractMethod(block.AccountAddress, sendBlock.Data, vm.latestSnapshotHeight); ok {
		// check quota
		quotaUsed := p.GetReceiveQuota(vm.gasTable)
		if quotaUsed > 0 {
			quotaTotal, _, err := quota.GetQuotaForBlock(
				db,
				block.AccountAddress,
				getStakeBeneficialAmount(db),
				block.Difficulty,
				vm.latestSnapshotHeight)
			if err != nil {
				return nil, noRetry, err
			}
			_, err = util.UseQuota(quotaTotal, quotaUsed)
			if err != nil {
				return nil, retry, err
			}
		}
		util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
		blockListToSend, err := p.DoReceive(db, block, sendBlock, vm)
		if err == nil {
			vm.updateBlock(db, block, err, quotaUsed, quotaUsed)
			vm.vmContext.sendBlockList = blockListToSend
			if db, err = vm.doSendBlockList(db); err == nil {
				block.Data = getReceiveCallData(db, err)
				return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, nil
			}
		}
		vm.revert(db)
		refundFlag := false
		refundData, needRefund := p.GetRefundData(sendBlock, vm.latestSnapshotHeight)
		refundFlag = doRefund(vm, db, block, sendBlock, refundData, needRefund, ledger.BlockTypeSendCall)
		vm.updateBlock(db, block, err, quotaUsed, quotaUsed)
		if refundFlag {
			var refundErr error
			if db, refundErr = vm.doSendBlockList(db); refundErr != nil {
				monitor.LogEvent("vm", "impossibleReceiveError")
				nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
				return nil, retry, err
			}
			block.Data = getReceiveCallData(db, err)
			return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, err
		}
		block.Data = getReceiveCallData(db, err)
		return &vm_db.VmAccountBlock{block, db}, noRetry, err
	}
	// check can make transaction
	quotaTotal, quotaAddition, err := quota.GetQuotaForBlock(
		db,
		block.AccountAddress,
		getStakeBeneficialAmount(db),
		block.Difficulty,
		vm.latestSnapshotHeight)
	util.DealWithErr(err)
	quotaLeft := quotaTotal
	cost, err := gasReceive(block, meta, vm.gasTable)
	if err != nil {
		return nil, noRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, retry, err
	}
	// add balance, create account if not exist
	util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
	// do transfer transaction if account code size is zero
	if meta == nil {
		qStakeUsed, qUsed := util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, nil)
		vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
		return &vm_db.VmAccountBlock{block, db}, noRetry, nil
	}
	// run code
	_, code := util.GetContractCode(db, &block.AccountAddress, nil)
	c := newContract(block, db, sendBlock, sendBlock.Data, quotaLeft)
	c.setCallCode(block.AccountAddress, code)
	_, err = c.run(vm)
	if err == nil {
		qStakeUsed, qUsed := util.CalcQuotaUsed(true, quotaTotal, quotaAddition, c.quotaLeft, nil)
		vm.updateBlock(db, block, err, qStakeUsed, qUsed)
		db, err = vm.doSendBlockList(db)
		if err == nil {
			block.Data = getReceiveCallData(db, err)
			return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, nil
		}
	}

	if err == util.ErrNoReliableStatus {
		return nil, retry, err
	}

	vm.revert(db)

	if err == util.ErrOutOfQuota {
		unConfirmedList := db.GetUnconfirmedBlocks(*db.Address())
		if len(unConfirmedList) > 0 {
			// Contract receive out of quota, current block is not first unconfirmed block, retry next snapshotBlock
			return nil, retry, err
		}
		// Contract receive out of quota, current block is first unconfirmed block, refund with no quota
		refundFlag := doRefund(vm, db, block, sendBlock, []byte{}, false, ledger.BlockTypeSendRefund)
		qStakeUsed, qUsed := util.CalcQuotaUsed(true, quotaTotal, quotaAddition, c.quotaLeft, err)
		vm.updateBlock(db, block, err, qStakeUsed, qUsed)
		if refundFlag {
			var refundErr error
			if db, refundErr = vm.doSendBlockList(db); refundErr != nil {
				monitor.LogEvent("vm", "impossibleReceiveError")
				nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
				return nil, retry, err
			}
			block.Data = getReceiveCallData(db, err)
			return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, err
		}
		block.Data = getReceiveCallData(db, err)
		return &vm_db.VmAccountBlock{block, db}, noRetry, err
	}

	refundFlag := doRefund(vm, db, block, sendBlock, []byte{}, false, ledger.BlockTypeSendRefund)
	qStakeUsed, qUsed := util.CalcQuotaUsed(true, quotaTotal, quotaAddition, c.quotaLeft, err)
	vm.updateBlock(db, block, err, qStakeUsed, qUsed)
	if refundFlag {
		var refundErr error
		if db, refundErr = vm.doSendBlockList(db); refundErr != nil {
			monitor.LogEvent("vm", "impossibleReceiveError")
			nodeConfig.log.Error("Impossible receive error", "err", refundErr, "fromhash", sendBlock.Hash)
			return nil, retry, err
		}
		block.Data = getReceiveCallData(db, err)
		return mergeReceiveBlock(db, block, vm.sendBlockList), noRetry, err
	}
	block.Data = getReceiveCallData(db, err)
	return &vm_db.VmAccountBlock{block, db}, noRetry, err
}

func doRefund(vm *VM, db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, refundData []byte, needRefund bool, refundBlockType byte) bool {
	refundFlag := false
	if sendBlock.Amount.Sign() > 0 && sendBlock.Fee.Sign() > 0 && sendBlock.TokenId == ledger.ViteTokenId {
		refundAmount := new(big.Int).Add(sendBlock.Amount, sendBlock.Fee)
		vm.vmContext.AppendBlock(
			util.MakeRequestBlock(
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
			vm.vmContext.AppendBlock(
				util.MakeRequestBlock(
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
			vm.vmContext.AppendBlock(
				util.MakeRequestBlock(
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
	if !refundFlag && needRefund {
		vm.vmContext.AppendBlock(
			util.MakeRequestBlock(
				block.AccountAddress,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				big.NewInt(0),
				ledger.ViteTokenId,
				refundData))
		refundFlag = true
	}
	return refundFlag
}

func (vm *VM) sendReward(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendReward"}, time.Now())

	// check can make transaction
	quotaLeft := quotaTotal
	if useQuota {
		var err error
		quotaLeft, err = useQuotaForSend(block, db, quotaLeft, vm.gasTable)
		if err != nil {
			return nil, err
		}
	}
	if block.AccountAddress != types.AddressGovernance &&
		block.AccountAddress != types.AddressAsset {
		return nil, util.ErrInvalidMethodParam
	}
	qStakeUsed, qUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, nil)
	vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
	return &vm_db.VmAccountBlock{block, db}, nil
}

func (vm *VM) sendRefund(db vm_db.VmDb, block *ledger.AccountBlock, useQuota bool, quotaTotal, quotaAddition uint64) (*vm_db.VmAccountBlock, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "sendRefund"}, time.Now())
	block.Fee = helper.Big0
	quotaLeft := quotaTotal
	if useQuota {
		var err error
		quotaLeft, err = useQuotaForSend(block, db, quotaLeft, vm.gasTable)
		if err != nil {
			return nil, err
		}
	}
	if !nodeConfig.canTransfer(db, block.TokenId, block.Amount, block.Fee) {
		return nil, util.ErrInsufficientBalance
	}
	util.SubBalance(db, &block.TokenId, block.Amount)
	qStakeUsed, qUsed := util.CalcQuotaUsed(useQuota, quotaTotal, quotaAddition, quotaLeft, nil)
	vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
	return &vm_db.VmAccountBlock{block, db}, nil
}

func (vm *VM) receiveReward(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveReward"}, time.Now())
	quotaTotal := uint64(0)
	quotaLeft := uint64(0)
	quotaAddition := uint64(0)
	var err error
	if !types.IsBuiltinContractAddrInUse(block.AccountAddress) {
		quotaTotal, quotaAddition, err = quota.GetQuotaForBlock(
			db,
			block.AccountAddress,
			getStakeBeneficialAmount(db),
			block.Difficulty,
			vm.latestSnapshotHeight)
		util.DealWithErr(err)
		quotaLeft = quotaTotal
		cost, err := gasReceive(block, meta, vm.gasTable)
		if err != nil {
			return nil, noRetry, err
		}
		quotaLeft, err = util.UseQuota(quotaLeft, cost)
		if err != nil {
			return nil, retry, err
		}
	}
	util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
	qStakeUsed, qUsed := util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, nil)
	vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
	return &vm_db.VmAccountBlock{block, db}, noRetry, nil
}

func (vm *VM) receiveRefund(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, meta *ledger.ContractMeta) (*vm_db.VmAccountBlock, bool, error) {
	defer monitor.LogTimerConsuming([]string{"vm", "receiveRefund"}, time.Now())
	// check can make transaction
	quotaTotal, quotaAddition, err := quota.GetQuotaForBlock(
		db,
		block.AccountAddress,
		getStakeBeneficialAmount(db),
		block.Difficulty,
		vm.latestSnapshotHeight)
	util.DealWithErr(err)
	quotaLeft := quotaTotal
	cost, err := gasReceive(block, meta, vm.gasTable)
	if err != nil {
		return nil, noRetry, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	if err != nil {
		return nil, retry, err
	}
	util.AddBalance(db, &sendBlock.TokenId, sendBlock.Amount)
	qStakeUsed, qUsed := util.CalcQuotaUsed(true, quotaTotal, quotaAddition, quotaLeft, nil)
	vm.updateBlock(db, block, nil, qStakeUsed, qUsed)
	return &vm_db.VmAccountBlock{block, db}, noRetry, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	_, code := util.GetContractCode(c.db, &contractAddr, vm.globalStatus)
	if len(code) > 0 {
		cNew := newContract(c.block, c.db, c.sendBlock, c.data, c.quotaLeft)
		cNew.setCallCode(contractAddr, code)
		ret, err = cNew.run(vm)
		c.quotaLeft = cNew.quotaLeft
		return ret, err
	}
	return nil, nil
}

func (vm *VM) updateBlock(db vm_db.VmDb, block *ledger.AccountBlock, err error, qStakeUsed, qUsed uint64) {
	block.Quota = qStakeUsed
	block.QuotaUsed = qUsed
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

// AppendBlock method append a send block to send block list of a contract receive block
func (context *vmContext) AppendBlock(block *ledger.AccountBlock) {
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

// OffChainReader read contract storage without tx
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
	vm.i = newInterpreter(sb.Height, true)
	vm.gasTable = util.QuotaTableByHeight(sb.Height)
	c := newContract(&ledger.AccountBlock{AccountAddress: *db.Address()}, db, &ledger.AccountBlock{ToAddress: *db.Address()}, data, offChainReaderGas)
	c.setCallCode(*db.Address(), code)
	return c.run(vm)
}

func getStakeBeneficialAmount(db vm_db.VmDb) *big.Int {
	stakeBeneficialAmount, err := db.GetStakeBeneficialAmount(db.Address())
	util.DealWithErr(err)
	return stakeBeneficialAmount
}

func useQuotaForSend(block *ledger.AccountBlock, db vm_db.VmDb, quotaLeft uint64, gasTable *util.QuotaTable) (uint64, error) {
	cost, err := gasSendCall(block, gasTable)
	if err != nil {
		return quotaLeft, err
	}
	quotaMultiplier, err := getQuotaMultiplierForS(db, block.ToAddress)
	if err != nil {
		return quotaLeft, err
	}
	cost, err = util.MultipleCost(cost, quotaMultiplier)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = util.UseQuota(quotaLeft, cost)
	return quotaLeft, err
}

func getContractMeta(db vm_db.VmDb) *ledger.ContractMeta {
	if !types.IsContractAddr(*db.Address()) {
		return nil
	}
	meta, err := db.GetContractMeta()
	util.DealWithErr(err)
	if meta == nil {
		util.DealWithErr(util.ErrContractNotExists)
	}
	return meta
}

// printDebugBlockInfo prints block info after execution.
func printDebugBlockInfo(block *ledger.AccountBlock, result *vm_db.VmAccountBlock, err error) {
	var str string
	if result != nil {
		if result.AccountBlock.IsSendBlock() {
			str = "{SelfAddr: " + result.AccountBlock.AccountAddress.String() + ", " +
				"ToAddr: " + result.AccountBlock.ToAddress.String() + ", " +
				"BlockType: " + strconv.FormatInt(int64(result.AccountBlock.BlockType), 10) + ", " +
				"Quota: " + strconv.FormatUint(result.AccountBlock.Quota, 10) + ", " +
				"QuotaUsed: " + strconv.FormatUint(result.AccountBlock.QuotaUsed, 10) + ", " +
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
						"Fee:" + sendBlock.Fee.String() + "},"
				}
				str = str + "]"
			}
			str = "{SelfAddr: " + result.AccountBlock.AccountAddress.String() + ", " +
				"FromHash: " + result.AccountBlock.FromBlockHash.String() + ", " +
				"BlockType: " + strconv.FormatInt(int64(result.AccountBlock.BlockType), 10) + ", " +
				"Quota: " + strconv.FormatUint(result.AccountBlock.Quota, 10) + ", " +
				"QuotaUsed: " + strconv.FormatUint(result.AccountBlock.QuotaUsed, 10) + ", " +
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
