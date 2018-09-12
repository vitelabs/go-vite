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
	"regexp"
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
	switch block.BlockType {
	case ledger.BlockTypeReceive, ledger.BlockTypeReceiveError:
		block.Data = sendBlock.Data
		block.Amount = sendBlock.Amount
		block.TokenId = sendBlock.TokenId
		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			return vm.receiveCreate(block, vm.calcCreateQuota(sendBlock.Fee))
		} else if sendBlock.BlockType == ledger.BlockTypeSendCall || sendBlock.BlockType == ledger.BlockTypeSendReward {
			return vm.receiveCall(block)
		} else if sendBlock.BlockType == ledger.BlockTypeSendMintage {
			return vm.receiveMintage(block)
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
	case ledger.BlockTypeSendMintage:
		quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress, block)
		block, err = vm.sendMintage(block, quotaTotal, quotaAddition)
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
	gid, _ := types.BytesToGid(block.Data[:10])
	if !isExistGid(vm.Db, gid) {
		return nil, ErrInvalidData
	}
	if !vm.canTransfer(block.AccountAddress, block.TokenId, block.Amount, block.Fee) {
		return nil, ErrInsufficientBalance
	}
	// create address
	contractAddr := createContractAddress(block.AccountAddress, block.Height, block.PrevHash, block.Data, block.SnapshotHash)

	if bytes.Equal(contractAddr.Bytes(), util.EmptyAddress.Bytes()) || vm.Db.IsAddressExisted(&contractAddr) {
		return nil, ErrContractAddressCreationFail
	}
	block.Fee = contractFee
	// sub balance and service fee
	vm.Db.SubBalance(&block.TokenId, block.Amount)
	if block.Fee != nil {
		vm.Db.SubBalance(ledger.ViteTokenId(), block.Fee)
	}
	vm.updateBlock(block, block.AccountAddress, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	block.ToAddress = contractAddr
	vm.Db.SetContractGid(&gid, &contractAddr, false)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block *ledger.AccountBlock, quotaTotal uint64) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	quotaLeft := quotaTotal
	if vm.Db.IsAddressExisted(&block.ToAddress) {
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
	block.Data = blockData[10:]
	defer func() { block.Data = blockData }()

	// init contract state and set contract code
	c := newContract(block.AccountAddress, block.ToAddress, block, quotaLeft, 0)
	c.setCallCode(block.ToAddress, block.Data)
	code, err := c.run(vm)
	if err == nil {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = useQuota(c.quotaLeft, codeCost)
		if err == nil {
			codeHash, _ := types.BytesToHash(code)
			gid, _ := types.BytesToGid(blockData[:10])
			vm.Db.SetContractCode(code)
			vm.updateBlock(block, block.ToAddress, nil, 0, codeHash.Bytes())
			err = vm.doSendBlockList(quotaTotal - block.Quota)
			if err == nil {
				vm.Db.SetContractGid(&gid, &block.ToAddress, true)
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
	if p, ok := getPrecompiledContract(block.ToAddress); ok {
		var err error
		block.Fee = p.createFee(vm, block)
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
	if _, ok := getPrecompiledContract(block.AccountAddress); ok {
		quota = 0
	} else {
		quota = quotaUsed(quotaTotal, quotaAddition, quotaLeft, 0, nil)
	}
	vm.updateBlock(block, block.AccountAddress, nil, quota, nil)
	return block, nil

}

func (vm *VM) receiveCall(block *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	if p, ok := getPrecompiledContract(block.ToAddress); ok {
		vm.blockList = []*ledger.AccountBlock{block}
		vm.Db.AddBalance(&block.TokenId, block.Amount)
		err := p.doReceive(vm, block)
		if err == nil {
			vm.updateBlock(block, block.ToAddress, err, 0, nil)
			err = vm.doSendBlockList(txGas)
			if err == nil {
				return vm.blockList, util.NoRetry, nil
			}
		}
		vm.revert()
		vm.updateBlock(block, block.ToAddress, err, 0, nil)
		return vm.blockList, util.NoRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition := vm.quotaLeft(block.ToAddress, block)
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
		code := vm.Db.GetContractCode(&block.ToAddress)
		if len(code) == 0 {
			vm.updateBlock(block, block.ToAddress, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
			return vm.blockList, util.NoRetry, nil
		}
		// run code
		c := newContract(block.AccountAddress, block.ToAddress, block, quotaLeft, quotaRefund)
		c.setCallCode(block.ToAddress, code)
		_, err = c.run(vm)
		if err == nil {
			vm.updateBlock(block, block.ToAddress, nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), nil)
			err = vm.doSendBlockList(quotaTotal - quotaAddition - block.Quota)
			if err == nil {
				return vm.blockList, util.NoRetry, nil
			}
		}

		vm.revert()
		vm.updateBlock(block, block.ToAddress, err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		return vm.blockList, err == ErrOutOfQuota, err
	}
}

func (vm *VM) sendMintage(block *ledger.AccountBlock, quotaTotal, quotaAddition uint64) (*ledger.AccountBlock, error) {
	if err := vm.checkToken(block.Data); err != nil {
		return nil, err
	}
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

	// calculate and check mintage fee
	mintageFee, err := calcMintageFee(block.Data)
	if err != nil {
		return nil, err
	}
	if !vm.canTransfer(block.AccountAddress, *ledger.ViteTokenId(), mintageFee, util.Big0) {
		return nil, ErrInsufficientBalance
	}

	// create tokenId and check collision
	tokenTypeId := createTokenId(block.AccountAddress, block.ToAddress, block.Height, block.PrevHash, block.SnapshotHash)
	if bytes.Equal(tokenTypeId.Bytes(), util.EmptyTokenTypeId.Bytes()) || vm.Db.GetToken(&tokenTypeId) != nil {
		return nil, ErrTokenIdCreationFail
	}

	// sub balance
	vm.Db.SubBalance(ledger.ViteTokenId(), mintageFee)
	block.TokenId = tokenTypeId
	block.Fee = mintageFee
	vm.updateBlock(block, block.AccountAddress, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return block, nil
}

func (vm *VM) receiveMintage(block *ledger.AccountBlock) (blockList []*ledger.AccountBlock, isRetry bool, err error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.ToAddress, block)
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
	param := new(VariableMintage)
	ABI_mintage.UnpackVariable(param, VariableNameMintage, block.Data)
	if vm.Db.GetToken(&block.TokenId) != nil {
		vm.Db.SetToken(&ledger.Token{TokenId: block.TokenId, TokenName: param.TokenName, TotalSupply: block.Amount, Decimals: int(param.Decimals)})
		vm.updateBlock(block, block.AccountAddress, ErrIdCollision, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, ErrIdCollision), nil)
		return vm.blockList, util.NoRetry, ErrIdCollision
	}
	vm.blockList = []*ledger.AccountBlock{block}
	vm.Db.AddBalance(&block.TokenId, block.Amount)
	vm.updateBlock(block, block.AccountAddress, nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return vm.blockList, util.NoRetry, nil
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
	vm.updateBlock(block, block.AccountAddress, nil, 0, nil)
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

func (vm *VM) updateBlock(block *ledger.AccountBlock, addr types.Address, err error, quota uint64, result []byte) {
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

		block.LogHash = *vm.Db.GetLogListHash()
		if err == ErrOutOfQuota {
			block.BlockType = ledger.BlockTypeReceiveError
		} else {
			block.BlockType = ledger.BlockTypeReceive
		}
		if len(vm.blockList) > 1 {
			block.SendBlockHashList = make([]types.Hash, len(vm.blockList))
			for i, sendBlock := range vm.blockList[1:] {
				block.SendBlockHashList[i] = sendBlock.GetComputeHash() // TODO calc send block hash
			}
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
		case ledger.BlockTypeSendMintage:
			vm.blockList[i+1], err = vm.sendCreate(block, quotaLeft, 0)
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

func (vm *VM) checkToken(data []byte) error {
	param := new(VariableMintage)
	err := ABI_mintage.UnpackVariable(param, VariableNameMintage, data)
	if err != nil {
		return ErrInvalidData
	}
	if param.Decimals < tokenDecimalsMin || param.Decimals > tokenDecimalsMax {
		return ErrInvalidData
	}
	if ok, _ := regexp.MatchString("^[a-zA-Z]+$", param.TokenName); !ok {
		return ErrInvalidData
	}
	return nil
}

func calcContractFee(data []byte) (*big.Int, error) {
	return contractFee, nil
}

func calcMintageFee(data []byte) (*big.Int, error) {
	return mintageFee, nil
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

func createTokenId(addr, owner types.Address, height uint64, prevHash, snapshotHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(addr.Bytes(), owner.Bytes(), new(big.Int).SetUint64(height).Bytes(), prevHash.Bytes(), snapshotHash.Bytes())
}

func isExistGid(db VmDatabase, gid types.Gid) bool {
	value := db.GetStorage(&AddressConsensusGroup, types.DataHash(gid.Bytes()).Bytes())
	return len(value) > 0
}
