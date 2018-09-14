package vm

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"regexp"
	"strconv"
	"time"
)

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressPledge, _         = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
	AddressMintage, _        = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5})
)

type precompiledContract struct {
	m   map[string]precompiledContractMethod
	abi abi.ABIContract
}
type precompiledContractMethod interface {
	getFee(vm *VM, block *ledger.AccountBlock) *big.Int
	doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error)
	doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error
}

var simpleContracts = map[types.Address]*precompiledContract{
	AddressRegister: {
		map[string]precompiledContractMethod{
			MethodNameRegister:           &pRegister{},
			MethodNameCancelRegister:     &pCancelRegister{},
			MethodNameReward:             &pReward{},
			MethodNameUpdateRegistration: &pUpdateRegistration{},
		},
		ABI_register,
	},
	AddressVote: {
		map[string]precompiledContractMethod{
			MethodNameVote:       &pVote{},
			MethodNameCancelVote: &pCancelVote{},
		},
		ABI_vote,
	},
	AddressPledge: {
		map[string]precompiledContractMethod{
			MethodNamePledge:       &pPledge{},
			MethodNameCancelPledge: &pCancelPledge{},
		},
		ABI_pledge,
	},
	AddressConsensusGroup: {
		map[string]precompiledContractMethod{
			MethodNameCreateConsensusGroup: &pCreateConsensusGroup{},
		},
		ABI_consensusGroup,
	},
	AddressMintage: {
		map[string]precompiledContractMethod{
			MethodNameMintage:             &pMintage{},
			MethodNameMintageCancelPledge: &pMintageCancelPledge{},
		},
		ABI_mintage,
	},
}

func isPrecompiledContractAddress(addr types.Address) bool {
	_, ok := simpleContracts[addr]
	return ok
}
func getPrecompiledContract(addr types.Address, methodSelector []byte) (precompiledContractMethod, bool) {
	p, ok := simpleContracts[addr]
	if ok {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, ok := p.m[method.Name]
			return c, ok
		}
	}
	return nil, false
}

type pRegister struct {
}

func (p *pRegister) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *pRegister) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(registerAmount) != 0 ||
		!IsViteToken(block.TokenId) {
		return quotaLeft, ErrInvalidData
	}

	param := new(ParamRegister)
	err = ABI_register.UnpackMethod(param, MethodNameRegister, block.Data)
	if err != nil || !isExistGid(vm.Db, param.Gid) ||
		!vm.Db.IsAddressExisted(&param.BeneficialAddr) ||
		!vm.Db.IsAddressExisted(&param.NodeAddr) ||
		!isUserAccount(vm.Db, param.NodeAddr) {
		return quotaLeft, ErrInvalidData
	}
	if len(vm.Db.GetStorage(&AddressRegister, getRegisterKey(param.Name, param.Gid))) > 0 {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pRegister) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamRegister)
	ABI_register.UnpackMethod(param, MethodNameRegister, block.Data)
	snapshotBlock := vm.Db.GetSnapshotBlock(&block.SnapshotHash)
	rewardHeight := snapshotBlock.Height
	key := getRegisterKey(param.Name, param.Gid)
	oldData := vm.Db.GetStorage(&block.AccountAddress, key)
	if len(oldData) > 0 {
		old := new(types.Registration)
		ABI_register.UnpackVariable(old, VariableNameRegistration, oldData)
		if old.Timestamp > 0 {
			// duplicate register
			return ErrInvalidData
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
	}
	registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, param.Name, param.NodeAddr, sendBlock.AccountAddress, param.BeneficialAddr, block.Amount, snapshotBlock.Timestamp.Unix(), rewardHeight, uint64(0))
	vm.Db.SetStorage(key, registerInfo)
	return nil
}

type pCancelRegister struct {
}

func (p *pCancelRegister) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *pCancelRegister) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamCancelRegister)
	err = ABI_register.UnpackMethod(param, MethodNameCancelRegister, block.Data)
	if err != nil || !isExistGid(vm.Db, param.Gid) {
		return quotaLeft, ErrInvalidData
	}

	key := getRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err = ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, key))
	if err != nil || !bytes.Equal(old.PledgeAddr.Bytes(), block.AccountAddress.Bytes()) ||
		old.Timestamp == 0 ||
		old.Timestamp+registerLockTime < vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp.Unix() {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pCancelRegister) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamCancelRegister)
	ABI_register.UnpackMethod(param, MethodNameCancelRegister, block.Data)

	key := getRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err := ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.AccountAddress, key))
	if err != nil || old.Timestamp == 0 {
		return ErrInvalidData
	}

	// update lock amount and loc start timestamp
	snapshotBlock := vm.Db.GetSnapshotBlock(&block.SnapshotHash)
	registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, param.Name, old.NodeAddr, old.PledgeAddr, old.BeneficialAddr, common.Big0, int64(0), old.RewardHeight, snapshotBlock.Height)
	vm.Db.SetStorage(key, registerInfo)
	// return locked ViteToken
	if old.Amount.Sign() > 0 {
		vm.blockList = append(vm.blockList, makeSendBlock(block, sendBlock.AccountAddress, ledger.BlockTypeSendCall, old.Amount, *ledger.ViteTokenId(), []byte{}))
	}
	return nil
}

type pReward struct {
}

func (p *pReward) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// get reward of generating snapshot block
func (p *pReward) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, rewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamReward)
	err = ABI_register.UnpackMethod(param, MethodNameReward, block.Data)
	if err != nil || !IsSnapshotGid(param.Gid) {
		return quotaLeft, ErrInvalidData
	}
	key := getRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err = ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, key))
	if err != nil || !bytes.Equal(block.AccountAddress.Bytes(), old.BeneficialAddr.Bytes()) {
		return quotaLeft, ErrInvalidData
	}
	// newRewardHeight := min(currentSnapshotHeight-50, userDefined, cancelSnapshotHeight)
	newRewardHeight := vm.Db.GetSnapshotBlock(&block.SnapshotHash).Height - rewardHeightLimit
	if param.EndHeight > 0 {
		newRewardHeight = helper.Min(newRewardHeight, param.EndHeight)
	}
	if old.CancelHeight > 0 {
		newRewardHeight = helper.Min(newRewardHeight, old.CancelHeight)
	}
	if newRewardHeight <= old.RewardHeight {
		return quotaLeft, ErrInvalidData
	}
	heightGap := newRewardHeight - old.RewardHeight
	if heightGap > rewardGapLimit {
		return quotaLeft, ErrInvalidData
	}

	count := heightGap
	quotaLeft, err = useQuota(quotaLeft, ((count+dbPageSize-1)/dbPageSize)*calcRewardGasPerPage)
	if err != nil {
		return quotaLeft, err
	}

	calcReward(vm, block.AccountAddress.Bytes(), old.RewardHeight, count, param.Amount)
	data, err := ABI_register.PackMethod(MethodNameReward, param.Gid, param.Name, newRewardHeight, old.RewardHeight, param.Amount)
	if err != nil {
		return quotaLeft, err
	}
	block.Data = data
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func calcReward(vm *VM, producer []byte, startHeight uint64, count uint64, reward *big.Int) {
	var rewardCount uint64
	for count > 0 {
		var list []*ledger.SnapshotBlock
		if count < dbPageSize {
			list = vm.Db.GetSnapshotBlocks(startHeight, count, true)
			count = 0
		} else {
			list = vm.Db.GetSnapshotBlocks(startHeight, dbPageSize, true)
			count = count - dbPageSize
			startHeight = startHeight + dbPageSize
		}
		for _, block := range list {
			if bytes.Equal(block.Producer.Bytes(), producer) {
				rewardCount++
			}
		}
	}
	reward.SetUint64(rewardCount)
	reward.Mul(rewardPerBlock, reward)
}
func (p *pReward) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamReward)
	ABI_register.UnpackMethod(param, MethodNameReward, block.Data)
	key := getRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err := ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.AccountAddress, key))
	if err != nil || old.RewardHeight != param.StartHeight || !bytes.Equal(sendBlock.AccountAddress.Bytes(), old.BeneficialAddr.Bytes()) {
		return ErrInvalidData
	}
	if old.CancelHeight > 0 {
		if param.EndHeight > old.CancelHeight {
			return ErrInvalidData
		} else if param.EndHeight == old.CancelHeight {
			// delete storage when register canceled and reward drained
			vm.Db.SetStorage(key, nil)
		} else {
			// get reward partly, update storage
			registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, old.Name, old.NodeAddr, old.PledgeAddr, old.BeneficialAddr, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
			vm.Db.SetStorage(key, registerInfo)
		}
	} else {
		registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, old.Name, old.NodeAddr, old.PledgeAddr, old.BeneficialAddr, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
		vm.Db.SetStorage(key, registerInfo)
	}

	if param.Amount.Sign() > 0 {
		// create reward and return
		vm.blockList = append(vm.blockList, makeSendBlock(block, sendBlock.AccountAddress, ledger.BlockTypeSendReward, param.Amount, *ledger.ViteTokenId(), []byte{}))
	}
	return nil
}

type pUpdateRegistration struct {
}

func (p *pUpdateRegistration) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// update registration info
func (p *pUpdateRegistration) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, updateRegistrationGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 {
		return quotaLeft, ErrInvalidData
	}

	param := new(ParamRegister)
	err = ABI_register.UnpackMethod(param, MethodNameUpdateRegistration, block.Data)
	if err != nil ||
		!vm.Db.IsAddressExisted(&param.BeneficialAddr) ||
		!vm.Db.IsAddressExisted(&param.NodeAddr) ||
		!isUserAccount(vm.Db, param.NodeAddr) {
		return quotaLeft, ErrInvalidData
	}
	old := new(types.Registration)
	err = ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&AddressRegister, getRegisterKey(param.Name, param.Gid)))
	if err != nil ||
		!bytes.Equal(old.PledgeAddr.Bytes(), block.AccountAddress.Bytes()) ||
		old.Timestamp == 0 ||
		(bytes.Equal(old.BeneficialAddr.Bytes(), param.BeneficialAddr.Bytes()) && bytes.Equal(old.NodeAddr.Bytes(), param.BeneficialAddr.Bytes())) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pUpdateRegistration) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamRegister)
	ABI_register.UnpackMethod(param, MethodNameUpdateRegistration, block.Data)
	key := getRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err := ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.AccountAddress, key))
	if err != nil || old.Timestamp == 0 {
		return ErrInvalidData
	}
	registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, old.Name, param.NodeAddr, old.PledgeAddr, param.BeneficialAddr, old.Amount, old.Timestamp, old.RewardHeight, old.CancelHeight)
	vm.Db.SetStorage(key, registerInfo)
	return nil
}

type pVote struct {
}

func (p *pVote) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// vote for a super node of a consensus group
func (p *pVote) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, voteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamVote)
	err = ABI_vote.UnpackMethod(param, MethodNameVote, block.Data)
	if err != nil || !isExistGid(vm.Db, param.Gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pVote) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamVote)
	ABI_vote.UnpackMethod(param, MethodNameVote, block.Data)
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := getVoteKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := ABI_vote.PackVariable(VariableNameVoteStatus, param.NodeName)
	vm.Db.SetStorage(locHash, voteStatus)
	return nil
}

type pCancelVote struct {
}

func (p *pCancelVote) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// cancel vote for a super node of a consensus group
func (p *pCancelVote) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	err = ABI_vote.UnpackMethod(gid, MethodNameCancelVote, block.Data)
	if err != nil || !isExistGid(vm.Db, *gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pCancelVote) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABI_vote.UnpackMethod(gid, MethodNameCancelVote, block.Data)
	locHash := getVoteKey(sendBlock.AccountAddress, *gid)
	vm.Db.SetStorage(locHash, nil)
	return nil
}

type pPledge struct{}

func (p *pPledge) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// pledge ViteToken for a beneficial to get quota
func (p *pPledge) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, pledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() == 0 ||
		!IsViteToken(block.TokenId) ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamPledge)
	err = ABI_pledge.UnpackMethod(param, MethodNamePledge, block.Data)
	if err != nil || !vm.Db.IsAddressExisted(&param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}

	if time.Unix(param.WithdrawTime-pledgeTime, 0).Before(*vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pPledge) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamPledge)
	ABI_pledge.UnpackMethod(param, MethodNamePledge, block.Data)
	// storage key for pledge beneficial: hash(beneficial)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes()).Bytes()
	// storage key for pledge: hash(owner, hash(beneficial))
	locHashPledge := types.DataHash(append(sendBlock.AccountAddress.Bytes(), locHashBeneficial...)).Bytes()
	oldPledgeData := vm.Db.GetStorage(&block.AccountAddress, locHashPledge)
	amount := new(big.Int)
	if len(oldPledgeData) > 0 {
		oldPledge := new(VariablePledgeInfo)
		ABI_pledge.UnpackVariable(oldPledge, VariableNamePledgeInfo, oldPledgeData)
		if param.WithdrawTime < oldPledge.WithdrawTime {
			return ErrInvalidData
		}
		amount = oldPledge.Amount
	}
	amount.Add(amount, block.Amount)
	pledgeInfo, _ := ABI_pledge.PackVariable(VariableNamePledgeInfo, amount, param.WithdrawTime)
	vm.Db.SetStorage(locHashPledge, pledgeInfo)

	// storage value for quota: quota amount(0:32)
	oldBeneficialData := vm.Db.GetStorage(&block.AccountAddress, locHashBeneficial)
	beneficialAmount := new(big.Int)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(VariablePledgeBeneficial)
		ABI_pledge.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, block.Amount)
	beneficialData, _ := ABI_pledge.PackVariable(VariableNamePledgeBeneficial, beneficialAmount)
	vm.Db.SetStorage(locHashBeneficial, beneficialData)
	return nil
}

type pCancelPledge struct{}

func (p *pCancelPledge) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

// cancel pledge ViteToken
func (p *pCancelPledge) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() > 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamCancelPledge)
	err = ABI_pledge.UnpackMethod(param, MethodNameCancelPledge, block.Data)
	if err != nil || !vm.Db.IsAddressExisted(&param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pCancelPledge) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamCancelPledge)
	ABI_pledge.UnpackMethod(param, MethodNameCancelPledge, block.Data)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes()).Bytes()
	locHashPledge := types.DataHash(append(sendBlock.AccountAddress.Bytes(), locHashBeneficial...)).Bytes()
	oldPledge := new(VariablePledgeInfo)
	err := ABI_pledge.UnpackVariable(oldPledge, VariableNamePledgeInfo, vm.Db.GetStorage(&block.AccountAddress, locHashPledge))
	if err != nil || time.Unix(oldPledge.WithdrawTime, 0).After(*vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(VariablePledgeBeneficial)
	err = ABI_pledge.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, vm.Db.GetStorage(&block.AccountAddress, locHashBeneficial))
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		vm.Db.SetStorage(locHashPledge, nil)
	} else {
		pledgeInfo, _ := ABI_pledge.PackVariable(VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawTime)
		vm.Db.SetStorage(locHashPledge, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		vm.Db.SetStorage(locHashBeneficial, nil)
	} else {
		pledgeBeneficial, _ := ABI_pledge.PackVariable(VariableNamePledgeBeneficial, oldBeneficial.Amount)
		vm.Db.SetStorage(locHashBeneficial, pledgeBeneficial)
	}

	// append refund block
	vm.blockList = append(vm.blockList, makeSendBlock(block, sendBlock.AccountAddress, ledger.BlockTypeSendCall, param.Amount, *ledger.ViteTokenId(), []byte{}))
	return nil
}

type pCreateConsensusGroup struct{}

func (p *pCreateConsensusGroup) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return new(big.Int).Set(createConsensusGroupFee)
}

// create consensus group
func (p *pCreateConsensusGroup) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, createConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(types.ConsensusGroupInfo)
	err = ABI_consensusGroup.UnpackMethod(param, MethodNameCreateConsensusGroup, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := p.checkCreateConsensusGroupData(vm, param); err != nil {
		return quotaLeft, err
	}
	// data: methodSelector(0:4) + gid(4:36) + ConsensusGroup
	gid := types.DataToGid(block.AccountAddress.Bytes(), new(big.Int).SetUint64(block.Height).Bytes(), block.PrevHash.Bytes(), block.SnapshotHash.Bytes())
	if isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}
	paramData, _ := ABI_consensusGroup.PackMethod(MethodNameCreateConsensusGroup, gid, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	block.Data = paramData
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func (p *pCreateConsensusGroup) checkCreateConsensusGroupData(vm *VM, param *types.ConsensusGroupInfo) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax {
		return ErrInvalidData
	}
	if err := p.checkCondition(vm, param.CountingRuleId, param.CountingRuleParam, "counting"); err != nil {
		return ErrInvalidData
	}
	if err := p.checkCondition(vm, param.RegisterConditionId, param.RegisterConditionParam, "register"); err != nil {
		return ErrInvalidData
	}
	if err := p.checkCondition(vm, param.VoteConditionId, param.VoteConditionParam, "vote"); err != nil {
		return ErrInvalidData
	}
	return nil
}
func (p *pCreateConsensusGroup) checkCondition(vm *VM, conditionId uint8, conditionParam []byte, conditionIdPrefix string) error {
	condition, ok := SimpleCountingRuleList[CountingRuleCode(conditionIdPrefix+strconv.Itoa(int(conditionId)))]
	if !ok {
		return ErrInvalidData
	}
	if ok := condition.checkParam(conditionParam, vm.Db); !ok {
		return ErrInvalidData
	}
	return nil
}
func (p *pCreateConsensusGroup) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(types.ConsensusGroupInfo)
	ABI_consensusGroup.UnpackMethod(param, MethodNameCreateConsensusGroup, block.Data)
	key := getConsensusGroupKey(param.Gid)
	if len(vm.Db.GetStorage(&block.AccountAddress, key)) > 0 {
		return ErrIdCollision
	}
	groupInfo, _ := ABI_consensusGroup.PackVariable(VariableNameConsensusGroupInfo, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	vm.Db.SetStorage(key, groupInfo)
	return nil
}

type CountingRuleCode string

const (
	CountingRuleOfBalance       CountingRuleCode = "counting1"
	RegisterConditionOfSnapshot CountingRuleCode = "register1"
	VoteConditionOfDefault      CountingRuleCode = "vote1"
	VoteConditionOfBalance      CountingRuleCode = "vote2"
)

type createConsensusGroupCondition interface {
	checkParam(param []byte, db VmDatabase) bool
}

var SimpleCountingRuleList = map[CountingRuleCode]createConsensusGroupCondition{
	CountingRuleOfBalance:       &countingRuleOfBalance{},
	RegisterConditionOfSnapshot: &registerConditionOfSnapshot{},
	VoteConditionOfDefault:      &voteConditionOfDefault{},
	VoteConditionOfBalance:      &voteConditionOfBalance{},
}

type countingRuleOfBalance struct{}

func (c countingRuleOfBalance) checkParam(param []byte, db VmDatabase) bool {
	v := new(types.TokenTypeId)
	err := ABI_consensusGroup.UnpackVariable(v, VariableNameConditionCounting1, param)
	if err != nil || contractsData.GetTokenById(db, *v) == nil {
		return false
	}
	return true
}

type registerConditionOfSnapshot struct{}

func (c registerConditionOfSnapshot) checkParam(param []byte, db VmDatabase) bool {
	v := new(VariableConditionRegister1)
	err := ABI_consensusGroup.UnpackVariable(v, VariableNameConditionRegister1, param)
	if err != nil || contractsData.GetTokenById(db, v.PledgeToken) == nil {
		return false
	}
	return true
}

type voteConditionOfDefault struct{}

func (c voteConditionOfDefault) checkParam(param []byte, db VmDatabase) bool {
	if len(param) != 0 {
		return false
	}
	return true
}

type voteConditionOfBalance struct{}

func (c voteConditionOfBalance) checkParam(param []byte, db VmDatabase) bool {
	v := new(VariableConditionVote2)
	err := ABI_consensusGroup.UnpackVariable(v, VariableNameConditionVote2, param)
	if err != nil || contractsData.GetTokenById(db, v.KeepToken) == nil {
		return false
	}
	return true
}

type pMintage struct{}

func (p *pMintage) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	if block.Amount.Cmp(mintagePledgeAmount) == 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Set(mintageFee)
}

func (p *pMintage) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, mintageGas)
	if err != nil {
		return quotaLeft, err
	}
	param := new(ParamMintage)
	err = ABI_mintage.UnpackMethod(param, MethodNameMintage, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err = checkToken(*param); err != nil {
		return quotaLeft, err
	}
	tokenId := types.CreateTokenTypeId(block.AccountAddress.Bytes(), new(big.Int).SetUint64(block.Height).Bytes(), block.PrevHash.Bytes(), block.SnapshotHash.Bytes())
	if contractsData.GetTokenById(vm.Db, tokenId) != nil {
		return quotaLeft, ErrIdCollision
	}
	block.Data, _ = ABI_mintage.PackMethod(MethodNameMintage, tokenId, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals)
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func checkToken(param ParamMintage) error {
	if param.Decimals < tokenDecimalsMin || param.Decimals > tokenDecimalsMax ||
		len(param.TokenName) == 0 || len(param.TokenName) > tokenNameLengthMax ||
		len(param.TokenSymbol) == 0 || len(param.TokenSymbol) > tokenSymbolLengthMax {
		return ErrInvalidData
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", param.TokenName); !ok {
		return ErrInvalidData
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_]+[ ]?)*[0-9a-zA-Z_]$", param.TokenSymbol); !ok {
		return ErrInvalidData
	}
	return nil
}
func (p *pMintage) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamMintage)
	ABI_mintage.UnpackMethod(param, MethodNameMintage, block.Data)
	locHash := helper.LeftPadBytes(param.TokenId.Bytes(), 32)
	if len(vm.Db.GetStorage(&block.AccountAddress, locHash)) > 0 {
		return ErrIdCollision
	}
	tokenInfo, _ := ABI_mintage.PackVariable(VariableNameMintage, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals, sendBlock.AccountAddress, sendBlock.Amount, vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp.Unix()+mintagePledgeTime)
	vm.Db.SetStorage(locHash, tokenInfo)
	vm.blockList = append(vm.blockList, makeSendBlock(block, sendBlock.AccountAddress, ledger.BlockTypeSendReward, param.TotalSupply, param.TokenId, []byte{}))
	return nil
}

type pMintageCancelPledge struct{}

func (p *pMintageCancelPledge) getFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

func (p *pMintageCancelPledge) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, mintageGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() > 0 {
		return quotaLeft, ErrInvalidData
	}
	tokenId := new(types.TokenTypeId)
	err = ABI_mintage.UnpackMethod(tokenId, MethodNameMintageCancelPledge, block.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}
	tokenInfo := contractsData.GetTokenById(vm.Db, *tokenId)
	if !bytes.Equal(tokenInfo.Owner.Bytes(), block.AccountAddress.Bytes()) || tokenInfo.PledgeAmount.Sign() == 0 || tokenInfo.Timestamp > vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp.Unix() {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pMintageCancelPledge) doReceive(vm *VM, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) error {
	tokenId := new(types.TokenTypeId)
	ABI_mintage.UnpackMethod(tokenId, MethodNameMintageCancelPledge, block.Data)
	storageKey := helper.LeftPadBytes(tokenId.Bytes(), types.HashSize)
	tokenInfo := new(types.TokenInfo)
	ABI_mintage.UnpackVariable(tokenInfo, VariableNameMintage, vm.Db.GetStorage(&block.AccountAddress, storageKey))
	newTokenInfo, _ := ABI_mintage.PackVariable(VariableNameMintage, tokenInfo.TokenName, tokenInfo.TokenSymbol, tokenInfo.TotalSupply, tokenInfo.Decimals, tokenInfo.Owner, big.NewInt(0), int64(0))
	vm.Db.SetStorage(storageKey, newTokenInfo)
	if tokenInfo.PledgeAmount.Sign() > 0 {
		vm.blockList = append(vm.blockList, makeSendBlock(block, tokenInfo.Owner, ledger.BlockTypeSendCall, tokenInfo.PledgeAmount, *ledger.ViteTokenId(), []byte{}))
	}
	return nil
}
func isUserAccount(db VmDatabase, addr types.Address) bool {
	return len(db.GetContractCode(&addr)) == 0
}
