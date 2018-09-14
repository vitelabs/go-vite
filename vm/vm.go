/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"bytes"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"sync/atomic"
)

type VMConfig struct {
	Debug bool
}

type VM struct {
	VMConfig
	Db VmDatabase

	abort          int32
	instructionSet [256]operation
	blockList      []*ledger.AccountBlock
	returnData     []byte
}

func NewVM(db VmDatabase) *VM {
	return &VM{Db: db, instructionSet: simpleInstructionSet}
}

func (vm *VM) Run(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	// TODO copy block
	switch block.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		// block data, amount, tokenId, fee is already changed to send block data by generator
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(block, sendBlock, vm.calcCreateQuota(block.Fee))
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(block, sendBlock)
		}
	case ledger.BlockTypeSendCreate:
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress, block)
		block, err = vm.sendCreate(block, quotaTotal, quotaAddition)
		if err != nil {
			return nil, util.NoRetry, err
		} else {
			return []*ledger.AccountBlock{block}, util.NoRetry, nil
		}
	case ledger.BlockTypeSendCall:
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress, block)
		block, err = vm.sendCall(block, quotaTotal, quotaAddition)
		if err != nil {
			return nil, util.NoRetry, err
		} else {
			return []*ledger.AccountBlock{block}, util.NoRetry, nil
		}
	}

	return nil, util.NoRetry, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block *ledger.AccountBlock, quotaTotal, quotaAddition uint64) (*ledger.AccountBlock, error) {
	// check can make transaction
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	contractFee, err := calcContractFee(block.Data)
	if err != nil {
		return nil, ErrInvalidData
	}
	gid, _ := types.BytesToGid(block.Data[:types.GidSize])
	if !isExistGid(vm.Db, gid) {
		return nil, ErrInvalidData
	}
	if !vm.canTransfer(block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
		return nil, ErrInsufficientBalance
	}
	// create address
	contractAddr := createContractAddress(block.AccountAddress, block.Height, block.PrevHash, block.Data, block.SnapshotHash)

	if vm.Db.IsAddressExisted(&contractAddr) {
		return nil, ErrContractAddressCreationFail
	}
	block.Fee = contractFee
	// sub balance and service fee
	vm.Db.SubBalance(&block.TokenId, block.Amount)
	if block.Fee != nil {
		vm.Db.SubBalance(ledger.ViteTokenId(), block.Fee)
	}
	vm.updateBlock(block, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	block.ToAddress = contractAddr
	vm.Db.SetContractGid(&gid, &contractAddr, false)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, quotaTotal uint64) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	quotaLeft := quotaTotal
	if vm.Db.IsAddressExisted(&block.AccountAddress) {
		return nil, util.NoRetry, ErrAddressCollision
	}
	// check can make transaction
	cost, err := intrinsicGasCost(nil, true)
	if err != nil {
		return nil, util.NoRetry, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, util.NoRetry, err
	}

	vm.blockList = []*ledger.AccountBlock{block}

	// create contract account and add balance
	vm.Db.AddBalance(&block.TokenId, block.Amount)

	blockData := block.Data
	block.Data = blockData[types.GidSize:]
	defer func() { block.Data = blockData }()

	// init contract state and set contract code
	c := newContract(sendBlock.AccountAddress, block.AccountAddress, block, quotaLeft, 0)
	c.setCallCode(block.AccountAddress, block.Data)
	code, err := c.run(vm)
	if err == nil {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = useQuota(c.quotaLeft, codeCost)
		if err == nil {
			codeHash, _ := types.BytesToHash(code)
			gid, _ := types.BytesToGid(blockData[:types.GidSize])
			vm.Db.SetContractCode(code)
			vm.updateBlock(block, nil, 0, codeHash.Bytes())
			err = vm.doSendBlockList(quotaTotal - block.Quota)
			if err == nil {
				vm.Db.SetContractGid(&gid, &block.AccountAddress, true)
				return vm.blockList, util.NoRetry, nil
			}
		}
	}

	vm.revert()
	return nil, util.NoRetry, err
}

func (vm *VM) sendCall(block *ledger.AccountBlock, quotaTotal, quotaAddition uint64) (*ledger.AccountBlock, error) {
	// check can make transaction
	quotaLeft := quotaTotal
	if p, ok := getPrecompiledContract(block.ToAddress, block.Data); ok {
		var err error
		block.Fee = p.getFee(vm, block)
		if !vm.canTransfer(block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
			return nil, ErrInsufficientBalance
		}
		quotaLeft, err = p.doSend(vm, block, quotaLeft)
		if err != nil {
			return nil, err
		}
		vm.Db.SubBalance(&block.TokenId, block.Amount)
		vm.Db.SubBalance(ledger.ViteTokenId(), block.Fee)
	} else {
		block.Fee = util.Big0
		cost, err := intrinsicGasCost(block.Data, false)
		if err != nil {
			return nil, err
		}
		quotaLeft, err = useQuota(quotaLeft, cost)
		if err != nil {
			return nil, err
		}
		if !vm.canTransfer(block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
			return nil, ErrInsufficientBalance
		}
		vm.Db.SubBalance(&block.TokenId, block.Amount)
	}
	var quota uint64
	if isPrecompiledContractAddress(block.AccountAddress) {
		quota = 0
	} else {
		quota = quotaUsed(quotaTotal, quotaAddition, quotaLeft, 0, nil)
	}
	vm.updateBlock(block, nil, quota, nil)
	return block, nil

}

func (vm *VM) receiveCall(block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	if p, ok := getPrecompiledContract(block.AccountAddress, block.Data); ok {
		vm.blockList = []*ledger.AccountBlock{block}
		vm.Db.AddBalance(&block.TokenId, block.Amount)
		err := p.doReceive(vm, block, sendBlock)
		if err == nil {
			vm.updateBlock(block, err, 0, nil)
			err = vm.doSendBlockList(txGas)
			if err == nil {
				return vm.blockList, util.NoRetry, nil
			}
		}
		vm.revert()
		vm.updateBlock(block, err, 0, nil)
		return vm.blockList, util.NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress, block)
		quotaLeft := quotaTotal
		quotaRefund := uint64(0)
		cost, err := intrinsicGasCost(nil, false)
		if err != nil {
			return nil, util.NoRetry, err
		}
		quotaLeft, err = useQuota(quotaLeft, cost)
		if err != nil {
			return nil, util.Retry, err
		}
		vm.blockList = []*ledger.AccountBlock{block}
		// add balance, create account if not exist
		vm.Db.AddBalance(&block.TokenId, block.Amount)
		// do transfer transaction if account code size is zero
		code := vm.Db.GetContractCode(&block.AccountAddress)
		if len(code) == 0 {
			vm.updateBlock(block, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
			return vm.blockList, util.NoRetry, nil
		}
		// run code
		c := newContract(sendBlock.AccountAddress, block.AccountAddress, block, quotaLeft, quotaRefund)
		c.setCallCode(block.AccountAddress, code)
		_, err = c.run(vm)
		if err == nil {
			vm.updateBlock(block, nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), nil)
			err = vm.doSendBlockList(quotaTotal - quotaAddition - block.Quota)
			if err == nil {
				return vm.blockList, util.NoRetry, nil
			}
		}

		vm.revert()
		vm.updateBlock(block, err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		return vm.blockList, err == ErrOutOfQuota, err
	}
}

func (vm *VM) sendReward(block *ledger.AccountBlock, quotaTotal, quotaAddition uint64) (*ledger.AccountBlock, error) {
	// check can make transaction
	quotaLeft := quotaTotal
	cost, err := intrinsicGasCost(block.Data, false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(block.AccountAddress.Bytes(), AddressRegister.Bytes()) {
		return nil, ErrInvalidData
	}
	vm.updateBlock(block, nil, 0, nil)
	return block, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	cNew := newContract(c.caller, c.address, c.block, c.quotaLeft, c.quotaRefund)
	cNew.setCallCode(contractAddr, vm.Db.GetContractCode(&contractAddr))
	ret, err = cNew.run(vm)
	c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
	return ret, err
}

func (vm *VM) calcCreateQuota(fee *big.Int) uint64 {
	quota := new(big.Int).Div(fee, quotaByCreateFeeAttov)
	if quota.IsUint64() {
		return util.Min(quotaLimitForTransaction, quota.Uint64())
	}
	return quotaLimitForTransaction
}

func (vm *VM) quotaLeft(addr types.Address, block *ledger.AccountBlock) (uint64, uint64) {
	// quotaInit = pledge amount of account address at current snapshot block status(attov) / quotaByPledge
	// get extra quota if calc PoW before a send transaction
	quotaInit := util.Min(new(big.Int).Div(GetPledgeAmount(vm.Db, addr), quotaByPledge).Uint64(), quotaLimit)
	quotaAddition := uint64(0)
	if len(block.Nonce) > 0 {
		quotaAddition = quotaForPoW
	}
	prevHash := block.PrevHash
	for {
		prevBlock := vm.Db.GetAccountBlockByHash(&prevHash)
		if prevBlock != nil && bytes.Equal(block.SnapshotHash.Bytes(), prevBlock.SnapshotHash.Bytes()) {
			// quick fail on a receive error block referencing to the same snapshot block
			// only one block gets extra quota when referencing to the same snapshot block
			if prevBlock.BlockType == ledger.BlockTypeReceiveError || (len(prevBlock.Nonce) > 0 && len(block.Nonce) > 0) {
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

func (vm *VM) updateBlock(block *ledger.AccountBlock, err error, quota uint64, result []byte) {
	block.Quota = quota
	block.StateHash = *vm.Db.GetStorageHash()
	if block.BlockType == ledger.BlockTypeReceive || block.BlockType == ledger.BlockTypeReceiveError {
		// data = fixed byte of execution result + result
		if err == nil {
			block.Data = append(DataResultPrefixSuccess, result...)
		} else if err == ErrExecutionReverted {
			block.Data = append(DataResultPrefixRevert, result...)
		} else {
			block.Data = append(DataResultPrefixFail, result...)
		}

		block.LogHash = vm.Db.GetLogListHash()
		if err == ErrOutOfQuota {
			block.BlockType = ledger.BlockTypeReceiveError
		} else {
			block.BlockType = ledger.BlockTypeReceive
		}
	}
}

func (vm *VM) doSendBlockList(quotaLeft uint64) (err error) {
	for i, block := range vm.blockList[1:] {
		switch block.BlockType {
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
		quotaLeft = quotaLeft - vm.blockList[i+1].Quota
	}
	return nil
}

func (vm *VM) revert() {
	vm.blockList = vm.blockList[:1]
	vm.returnData = nil
	vm.Db.Reset()
}

func (vm *VM) canTransfer(addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
	if feeAmount == nil || feeAmount.Sign() == 0 {
		return tokenAmount.Cmp(vm.Db.GetBalance(&addr, &tokenTypeId)) <= 0
	} else if util.IsViteToken(tokenTypeId) {
		balance := new(big.Int).Add(tokenAmount, feeAmount)
		return balance.Cmp(vm.Db.GetBalance(&addr, &tokenTypeId)) <= 0
	} else {
		return tokenAmount.Cmp(vm.Db.GetBalance(&addr, &tokenTypeId)) <= 0 && feeAmount.Cmp(vm.Db.GetBalance(&addr, ledger.ViteTokenId())) <= 0
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
			return quotaTotal - quotaLeft - quotaAddition - util.Min(quotaRefund, (quotaTotal-quotaAddition-quotaLeft)/2)
		}
	}
}

func createContractAddress(addr types.Address, height uint64, prevHash types.Hash, code []byte, snapshotHash types.Hash) types.Address {
	return types.CreateContractAddress(addr.Bytes(), new(big.Int).SetUint64(height).Bytes(), prevHash.Bytes(), code, snapshotHash.Bytes())
}

func isExistGid(db VmDatabase, gid types.Gid) bool {
	value := db.GetStorage(&AddressConsensusGroup, getConsensusGroupKey(gid))
	return len(value) > 0
}

func makeSendBlock(block *ledger.AccountBlock, toAddress types.Address, blockType byte, amount *big.Int, tokenId types.TokenTypeId, data []byte) *ledger.AccountBlock {
	return &ledger.AccountBlock{AccountAddress: block.AccountAddress, ToAddress: toAddress, BlockType: blockType, Amount: amount, TokenId: tokenId, Height: block.Height + 1, SnapshotHash: block.SnapshotHash, Data: data}
}
