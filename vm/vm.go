/**
Package vm implements the vite virtual machine
*/
package vm

import (
	"bytes"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"regexp"
	"sync/atomic"
)

type VMConfig struct {
	Debug bool
}

type Log struct {
	// list of topics provided by the contract
	Topics []types.Hash
	// supplied by the contract, usually ABI-encoded
	Data []byte
}

type VM struct {
	VMConfig
	Db          VmDatabase
	createBlock CreateAccountBlockFunc

	abort int32
	//intPool        *intPool
	instructionSet [256]operation
	blockList      []VmAccountBlock
	returnData     []byte
}

func NewVM(db VmDatabase, createBlockFunc CreateAccountBlockFunc) *VM {
	return &VM{Db: db, createBlock: createBlockFunc, instructionSet: simpleInstructionSet}
}

func (vm *VM) Run(block VmAccountBlock) (blockList []VmAccountBlock, isRetry bool, err error) {
	switch block.BlockType() {
	case BlockTypeReceive, BlockTypeReceiveError:
		sendBlock := vm.Db.AccountBlock(block.AccountAddress(), block.FromBlockHash())
		block.SetData(sendBlock.Data())
		block.SetAmount(sendBlock.Amount())
		block.SetTokenId(sendBlock.TokenId())
		if sendBlock.BlockType() == BlockTypeSendCreate {
			return vm.receiveCreate(block, vm.calcCreateQuota(sendBlock.CreateFee()))
		} else if sendBlock.BlockType() == BlockTypeSendCall {
			return vm.receiveCall(block)
		} else if sendBlock.BlockType() == BlockTypeSendMintage {
			return vm.receiveMintage(block)
		}
	case BlockTypeSendCreate:
		block, err = vm.sendCreate(block)
		if err != nil {
			return nil, noRetry, err
		} else {
			return []VmAccountBlock{block}, noRetry, nil
		}
	case BlockTypeSendCall:
		block, err = vm.sendCall(block)
		if err != nil {
			return nil, noRetry, err
		} else {
			return []VmAccountBlock{block}, noRetry, nil
		}
	case BlockTypeSendMintage:
		block, err = vm.sendMintage(block)
		if err != nil {
			return nil, noRetry, err
		} else {
			return []VmAccountBlock{block}, noRetry, nil
		}
	}
	return nil, noRetry, errors.New("transaction type not supported")
}

func (vm *VM) Cancel() {
	atomic.StoreInt32(&vm.abort, 1)
}

// send contract create transaction, create address, sub balance and service fee
func (vm *VM) sendCreate(block VmAccountBlock) (VmAccountBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data(), false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if !checkContractFee(block.CreateFee()) {
		return nil, ErrInvalidContractFee
	}
	if !vm.canTransfer(block.AccountAddress(), block.TokenId(), block.Amount(), block.CreateFee()) {
		return nil, ErrInsufficientBalance
	}
	// create address
	contractAddr := createContractAddress(block.AccountAddress(), block.Height(), block.PrevHash(), block.Data(), block.SnapshotHash())

	if bytes.Equal(contractAddr.Bytes(), emptyAddress.Bytes()) || vm.Db.IsExistAddress(contractAddr) {
		return nil, ErrContractAddressCreationFail
	}
	// sub balance and service fee
	vm.Db.SubBalance(block.AccountAddress(), block.TokenId(), block.Amount())
	vm.Db.SubBalance(block.AccountAddress(), viteTokenTypeId, block.CreateFee())
	vm.updateBlock(block, block.AccountAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	block.SetToAddress(contractAddr)
	return block, nil
}

// receive contract create transaction, create contract account, run initialization code, set contract code, do send blocks
func (vm *VM) receiveCreate(block VmAccountBlock, quotaLeft uint64) (blockList []VmAccountBlock, isRetry bool, err error) {
	if vm.Db.IsExistAddress(block.ToAddress()) {
		return nil, noRetry, ErrAddressCollision
	}
	// check can make transaction
	cost, err := intrinsicGasCost(nil, true)
	if err != nil {
		return nil, noRetry, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, noRetry, err
	}

	vm.blockList = []VmAccountBlock{block}

	// create contract account and add balance
	vm.Db.AddBalance(block.ToAddress(), block.TokenId(), block.Amount())

	// init contract state and set contract code
	c := newContract(block.AccountAddress(), block.ToAddress(), block, quotaLeft, 0)
	c.setCallCode(block.ToAddress(), block.Data())
	code, err := c.run(vm)
	if err == nil {
		codeCost := uint64(len(code)) * contractCodeGas
		c.quotaLeft, err = useQuota(c.quotaLeft, codeCost)
		if err == nil {
			codeHash, _ := types.BytesToHash(code)
			vm.Db.SetContractCode(block.ToAddress(), code)
			vm.updateBlock(block, block.ToAddress(), nil, 0, codeHash.Bytes())
			err = vm.doSendBlockList()
			if err == nil {
				return vm.blockList, noRetry, nil
			}
		}
	}

	vm.revert()
	vm.updateBlock(block, block.ToAddress(), err, 0, nil)
	return vm.blockList, noRetry, err
}

func (vm *VM) sendCall(block VmAccountBlock) (VmAccountBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	var cost uint64
	var err error
	if _, ok := getPrecompiledContract(block.ToAddress()); ok {
		cost, err = intrinsicGasCost(nil, false)
	} else {
		cost, err = intrinsicGasCost(block.Data(), false)
	}
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}
	if !vm.canTransfer(block.AccountAddress(), block.TokenId(), block.Amount(), big0) {
		return nil, ErrInsufficientBalance
	}

	if p, ok := getPrecompiledContract(block.ToAddress()); ok {
		quotaLeft, quotaRefund, err = p.doSend(vm, block, quotaLeft, quotaRefund)
		if err != nil {
			return nil, err
		}
	}
	// sub balance
	vm.Db.SubBalance(block.AccountAddress(), block.TokenId(), block.Amount())
	vm.updateBlock(block, block.AccountAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return block, nil
}

func (vm *VM) receiveCall(block VmAccountBlock) (blockList []VmAccountBlock, isRetry bool, err error) {
	if p, ok := getPrecompiledContract(block.ToAddress()); ok {
		vm.blockList = []VmAccountBlock{block}
		vm.Db.AddBalance(block.ToAddress(), block.TokenId(), block.Amount())
		err := p.doReceive(vm, block)
		if err != nil {
			vm.updateBlock(block, block.ToAddress(), err, 0, nil)
			// TODO use quota left to send block list
			err = vm.doSendBlockList()
			if err == nil {
				return vm.blockList, noRetry, nil
			}
		}
		vm.revert()
		vm.updateBlock(block, block.ToAddress(), err, 0, nil)
		return vm.blockList, noRetry, err
	} else {
		// check can make transaction
		quotaTotal, quotaAddition := vm.quotaLeft(block.ToAddress(), block)
		quotaLeft := quotaTotal
		quotaRefund := uint64(0)
		cost, err := intrinsicGasCost(nil, false)
		if err != nil {
			return nil, noRetry, err
		}
		quotaLeft, err = useQuota(quotaLeft, cost)
		if err != nil {
			return nil, retry, err
		}
		vm.blockList = []VmAccountBlock{block}
		// add balance, create account if not exist
		vm.Db.AddBalance(block.ToAddress(), block.TokenId(), block.Amount())
		// do transfer transaction if account code size is zero
		code := vm.Db.ContractCode(block.ToAddress())
		if len(code) == 0 {
			vm.updateBlock(block, block.ToAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
			return vm.blockList, noRetry, nil
		}
		// run code
		c := newContract(block.AccountAddress(), block.ToAddress(), block, quotaLeft, quotaRefund)
		c.setCallCode(block.ToAddress(), code)
		_, err = c.run(vm)
		if err == nil {
			vm.updateBlock(block, block.ToAddress(), nil, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, nil), nil)
			err = vm.doSendBlockList()
			if err == nil {
				return vm.blockList, noRetry, nil
			}
		}

		vm.revert()
		vm.updateBlock(block, block.ToAddress(), err, quotaUsed(quotaTotal, quotaAddition, c.quotaLeft, c.quotaRefund, err), nil)
		return vm.blockList, err == ErrOutOfQuota, err
	}
}

func (vm *VM) sendMintage(block VmAccountBlock) (VmAccountBlock, error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.AccountAddress(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(block.Data(), false)
	if err != nil {
		return nil, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, err
	}

	// calculate and check mintage fee
	mintageFee, err := calcMintageFee(block.Data())
	if err != nil {
		return nil, err
	}
	if !vm.canTransfer(block.AccountAddress(), viteTokenTypeId, mintageFee, big0) {
		return nil, ErrInsufficientBalance
	}

	// create tokenId and check collision
	tokenTypeId := createTokenId(block.AccountAddress(), block.ToAddress(), block.Height(), block.PrevHash(), block.SnapshotHash())
	if bytes.Equal(tokenTypeId.Bytes(), emptyTokenTypeId.Bytes()) {
		return nil, ErrTokenIdCreationFail
	}
	if err = vm.checkAndCreateToken(tokenTypeId, block.ToAddress(), block.Amount(), block.Data()); err != nil {
		return nil, err
	}

	// sub balance
	vm.Db.SubBalance(block.AccountAddress(), viteTokenTypeId, mintageFee)
	block.SetTokenId(tokenTypeId)
	block.SetCreateFee(mintageFee)
	vm.updateBlock(block, block.AccountAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return block, nil
}

func (vm *VM) receiveMintage(block VmAccountBlock) (blockList []VmAccountBlock, isRetry bool, err error) {
	// check can make transaction
	quotaTotal, quotaAddition := vm.quotaLeft(block.ToAddress(), block)
	quotaLeft := quotaTotal
	quotaRefund := uint64(0)
	cost, err := intrinsicGasCost(nil, false)
	if err != nil {
		return nil, noRetry, err
	}
	quotaLeft, err = useQuota(quotaLeft, cost)
	if err != nil {
		return nil, retry, err
	}
	vm.blockList = []VmAccountBlock{block}
	vm.Db.AddBalance(block.ToAddress(), block.TokenId(), block.Amount())
	vm.updateBlock(block, block.AccountAddress(), nil, quotaUsed(quotaTotal, quotaAddition, quotaLeft, quotaRefund, nil), nil)
	return vm.blockList, noRetry, nil
}

func (vm *VM) delegateCall(contractAddr types.Address, data []byte, c *contract) (ret []byte, err error) {
	cNew := newContract(c.caller, c.address, c.block, c.quotaLeft, c.quotaRefund)
	cNew.setCallCode(contractAddr, vm.Db.ContractCode(contractAddr))
	ret, err = cNew.run(vm)
	c.quotaLeft, c.quotaRefund = cNew.quotaLeft, cNew.quotaRefund
	return ret, err
}

func (vm *VM) calcCreateQuota(fee *big.Int) uint64 {
	// TODO calculate quota for create contract receive transaction
	return quotaLimitForTransaction
}

func (vm *VM) quotaLeft(addr types.Address, block VmAccountBlock) (quotaInit, quotaAddition uint64) {
	// TODO calculate quota, use max for test
	// TODO calculate quota addition
	quotaInit = quotaLimit
	quotaAddition = 0
	for _, block := range vm.blockList {
		if quotaInit <= block.Quota() {
			return 0, 0
		} else {
			quotaInit = quotaInit - block.Quota()
		}
	}
	prevHash := block.PrevHash()
	if len(vm.blockList) > 0 {
		prevHash = vm.blockList[0].PrevHash()
	}
	for {
		prevBlock := vm.Db.AccountBlock(addr, prevHash)
		if prevBlock != nil && bytes.Equal(block.SnapshotHash().Bytes(), prevBlock.SnapshotHash().Bytes()) {
			quotaInit = quotaInit - prevBlock.Quota()
			prevHash = prevBlock.PrevHash()
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

func (vm *VM) updateBlock(block VmAccountBlock, addr types.Address, err error, quota uint64, result []byte) {
	block.SetQuota(quota)
	if block.BlockType() == BlockTypeReceive || block.BlockType() == BlockTypeReceiveError {
		// TODO data = fixed byte of err + result
		block.SetData(result)
		block.SetStateHash(vm.Db.StorageHash(addr))
		block.SetLogListHash(vm.Db.LogListHash())
		if err == ErrOutOfQuota {
			block.SetBlockType(BlockTypeReceiveError)
		} else {
			block.SetBlockType(BlockTypeReceive)
		}
		if len(vm.blockList) > 1 {
			for _, sendBlock := range vm.blockList[1:] {
				block.AppendSendBlockHash(sendBlock.SummaryHash())
			}
		}
	}
}

func (vm *VM) doSendBlockList() (err error) {
	for i, block := range vm.blockList[1:] {
		if !bytes.Equal(block.ToAddress().Bytes(), emptyAddress.Bytes()) {
			vm.blockList[i], err = vm.sendCall(block)
			if err != nil {
				return err
			}
		} else {
			vm.blockList[i], err = vm.sendCreate(block)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *VM) revert() {
	vm.blockList = vm.blockList[:1]
	vm.returnData = nil
	vm.Db.Rollback()
}

func (vm *VM) canTransfer(addr types.Address, tokenTypeId types.TokenTypeId, tokenAmount *big.Int, feeAmount *big.Int) bool {
	if feeAmount.Sign() == 0 {
		return tokenAmount.Cmp(vm.Db.Balance(addr, tokenTypeId)) <= 0
	} else if bytes.Equal(tokenTypeId.Bytes(), viteTokenTypeId.Bytes()) {
		balance := new(big.Int).Add(tokenAmount, feeAmount)
		return balance.Cmp(vm.Db.Balance(addr, tokenTypeId)) <= 0
	} else {
		return tokenAmount.Cmp(vm.Db.Balance(addr, tokenTypeId)) <= 0 && feeAmount.Cmp(vm.Db.Balance(addr, viteTokenTypeId)) <= 0
	}
}

func (vm *VM) checkAndCreateToken(tokenId types.TokenTypeId, owner types.Address, amount *big.Int, data []byte) error {
	if !bytes.Equal(data[0:32], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 64}) {
		return ErrInvalidTokenData
	}
	start := big.NewInt(32)
	decimals := new(big.Int).SetBytes(getDataBig(data, start, big32)).Uint64()
	if decimals <= tokenDecimalsMin || decimals > tokenDecimalsMax {
		return ErrInvalidTokenData
	}
	start.Add(start, big32)
	length := int(new(big.Int).SetBytes(getDataBig(data, start, big32)).Uint64())
	if length > tokenNameLengthLimit {
		return ErrInvalidTokenData
	}
	start.Add(start, big32)
	tokenName := hexToString(data[start.Int64():])
	if len(tokenName) != length {
		return ErrInvalidTokenData
	}
	if ok, _ := regexp.MatchString("^[a-zA-Z]+$", tokenName); !ok {
		return ErrInvalidTokenData
	}
	// TODO create token at receive mintage
	if vm.Db.CreateToken(tokenId, tokenName, owner, amount, decimals) {
		return nil
	} else {
		return ErrTokenIdCollision
	}
}

func checkContractFee(fee *big.Int) bool {
	return ContractFeeMin.Cmp(fee) <= 0 && ContractFeeMax.Cmp(fee) >= 0
}

func calcMintageFee(data []byte) (*big.Int, error) {
	// TODO calculate mintage fee
	return big0, nil
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
			return quotaTotal - quotaLeft - quotaAddition - min(quotaRefund, (quotaTotal-quotaAddition-quotaLeft)/2)
		}
	}
}

func createContractAddress(addr types.Address, height *big.Int, prevHash types.Hash, code []byte, snapshotHash types.Hash) types.Address {
	return types.CreateContractAddress(addr.Bytes(), height.Bytes(), prevHash.Bytes(), code, snapshotHash.Bytes())
}

func createTokenId(addr, owner types.Address, height *big.Int, prevHash, snapshotHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(addr.Bytes(), owner.Bytes(), height.Bytes(), prevHash.Bytes(), snapshotHash.Bytes())
}
