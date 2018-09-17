package vm

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vitelabs/go-vite/abi"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"regexp"
	"time"
)

type precompiledContract struct {
	m   map[string]precompiledContractMethod
	abi abi.ABIContract
}
type precompiledContractMethod interface {
	getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error)
	doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error)
	doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error
}

var simpleContracts = map[types.Address]*precompiledContract{
	contracts.AddressRegister: {
		map[string]precompiledContractMethod{
			contracts.MethodNameRegister:           &pRegister{},
			contracts.MethodNameCancelRegister:     &pCancelRegister{},
			contracts.MethodNameReward:             &pReward{},
			contracts.MethodNameUpdateRegistration: &pUpdateRegistration{},
		},
		contracts.ABI_register,
	},
	contracts.AddressVote: {
		map[string]precompiledContractMethod{
			contracts.MethodNameVote:       &pVote{},
			contracts.MethodNameCancelVote: &pCancelVote{},
		},
		contracts.ABI_vote,
	},
	contracts.AddressPledge: {
		map[string]precompiledContractMethod{
			contracts.MethodNamePledge:       &pPledge{},
			contracts.MethodNameCancelPledge: &pCancelPledge{},
		},
		contracts.ABI_pledge,
	},
	contracts.AddressConsensusGroup: {
		map[string]precompiledContractMethod{
			contracts.MethodNameCreateConsensusGroup: &pCreateConsensusGroup{},
		},
		contracts.ABI_consensusGroup,
	},
	contracts.AddressMintage: {
		map[string]precompiledContractMethod{
			contracts.MethodNameMintage:             &pMintage{},
			contracts.MethodNameMintageCancelPledge: &pMintageCancelPledge{},
		},
		contracts.ABI_mintage,
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

func (p *pRegister) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *pRegister) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamRegister)
	err = contracts.ABI_register.UnpackMethod(param, contracts.MethodNameRegister, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}

	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, RegisterConditionPrefix); !ok {
		return quotaLeft, ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, contracts.MethodNameRegister) {
		return quotaLeft, ErrInvalidData
	}

	if len(block.VmContext.GetStorage(&contracts.AddressRegister, contracts.GetRegisterKey(param.Name, param.Gid))) > 0 {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pRegister) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamRegister)
	contracts.ABI_register.UnpackMethod(param, contracts.MethodNameRegister, block.AccountBlock.Data)
	snapshotBlock := block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash)
	rewardHeight := snapshotBlock.Height
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	oldData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)
	if len(oldData) > 0 {
		old := new(contracts.Registration)
		contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, oldData)
		if old.Timestamp > 0 {
			// duplicate register
			return ErrInvalidData
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
	}
	registerInfo, _ := contracts.ABI_register.PackVariable(contracts.VariableNameRegistration, param.Name, param.NodeAddr, sendBlock.AccountAddress, param.BeneficialAddr, block.AccountBlock.Amount, snapshotBlock.Timestamp.Unix(), rewardHeight, uint64(0))
	block.VmContext.SetStorage(key, registerInfo)
	return nil
}

type pCancelRegister struct {
}

func (p *pCancelRegister) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *pCancelRegister) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamCancelRegister)
	err = contracts.ABI_register.UnpackMethod(param, contracts.MethodNameCancelRegister, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}

	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, RegisterConditionPrefix); !ok {
		return quotaLeft, ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, contracts.MethodNameCancelRegister) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pCancelRegister) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamCancelRegister)
	contracts.ABI_register.UnpackMethod(param, contracts.MethodNameCancelRegister, block.AccountBlock.Data)

	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err := contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || old.Timestamp == 0 {
		return ErrInvalidData
	}

	// update lock amount and loc start timestamp
	snapshotBlock := block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash)
	registerInfo, _ := contracts.ABI_register.PackVariable(contracts.VariableNameRegistration, param.Name, old.NodeAddr, old.PledgeAddr, old.BeneficialAddr, common.Big0, int64(0), old.RewardHeight, snapshotBlock.Height)
	block.VmContext.SetStorage(key, registerInfo)
	// return locked ViteToken
	if old.Amount.Sign() > 0 {
		vm.blockList = append(vm.blockList, &vm_context.VmAccountBlock{makeSendBlock(block.AccountBlock, sendBlock.AccountAddress, ledger.BlockTypeSendCall, old.Amount, *ledger.ViteTokenId(), []byte{}), nil})
	}
	return nil
}

type pReward struct {
}

func (p *pReward) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// get reward of generating snapshot block
func (p *pReward) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, rewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ParamReward)
	err = contracts.ABI_register.UnpackMethod(param, contracts.MethodNameReward, block.AccountBlock.Data)
	if err != nil || !IsSnapshotGid(param.Gid) {
		return quotaLeft, ErrInvalidData
	}
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err = contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.ToAddress, key))
	if err != nil || !bytes.Equal(block.AccountBlock.AccountAddress.Bytes(), old.BeneficialAddr.Bytes()) {
		return quotaLeft, ErrInvalidData
	}
	// newRewardHeight := min(currentSnapshotHeight-50, userDefined, cancelSnapshotHeight)
	newRewardHeight := block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash).Height - rewardHeightLimit
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

	calcReward(block.VmContext, block.AccountBlock.AccountAddress.Bytes(), old.RewardHeight, count, param.Amount)
	data, err := contracts.ABI_register.PackMethod(contracts.MethodNameReward, param.Gid, param.Name, newRewardHeight, old.RewardHeight, param.Amount)
	if err != nil {
		return quotaLeft, err
	}
	block.AccountBlock.Data = data
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func calcReward(db vmctxt_interface.VmDatabase, producer []byte, startHeight uint64, count uint64, reward *big.Int) {
	var rewardCount uint64
	for count > 0 {
		var list []*ledger.SnapshotBlock
		if count < dbPageSize {
			list = db.GetSnapshotBlocks(startHeight, count, true, false)
			count = 0
		} else {
			list = db.GetSnapshotBlocks(startHeight, dbPageSize, true, false)
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
func (p *pReward) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamReward)
	contracts.ABI_register.UnpackMethod(param, contracts.MethodNameReward, block.AccountBlock.Data)
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err := contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || old.RewardHeight != param.StartHeight || !bytes.Equal(sendBlock.AccountAddress.Bytes(), old.BeneficialAddr.Bytes()) {
		return ErrInvalidData
	}
	if old.CancelHeight > 0 {
		if param.EndHeight > old.CancelHeight {
			return ErrInvalidData
		} else if param.EndHeight == old.CancelHeight {
			// delete storage when register canceled and reward drained
			block.VmContext.SetStorage(key, nil)
		} else {
			// get reward partly, update storage
			registerInfo, _ := contracts.ABI_register.PackVariable(contracts.VariableNameRegistration, old.Name, old.NodeAddr, old.PledgeAddr, old.BeneficialAddr, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
			block.VmContext.SetStorage(key, registerInfo)
		}
	} else {
		registerInfo, _ := contracts.ABI_register.PackVariable(contracts.VariableNameRegistration, old.Name, old.NodeAddr, old.PledgeAddr, old.BeneficialAddr, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
		block.VmContext.SetStorage(key, registerInfo)
	}

	if param.Amount.Sign() > 0 {
		// create reward and return
		vm.blockList = append(vm.blockList, &vm_context.VmAccountBlock{makeSendBlock(block.AccountBlock, sendBlock.AccountAddress, ledger.BlockTypeSendReward, param.Amount, *ledger.ViteTokenId(), []byte{}), nil})
	}
	return nil
}

type pUpdateRegistration struct {
}

func (p *pUpdateRegistration) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// update registration info
func (p *pUpdateRegistration) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, updateRegistrationGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamRegister)
	err = contracts.ABI_register.UnpackMethod(param, contracts.MethodNameUpdateRegistration, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}

	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, RegisterConditionPrefix); !ok {
		return quotaLeft, ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, contracts.MethodNameUpdateRegistration) {
		return quotaLeft, ErrInvalidData
	}

	return quotaLeft, nil
}
func (p *pUpdateRegistration) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamRegister)
	contracts.ABI_register.UnpackMethod(param, contracts.MethodNameUpdateRegistration, block.AccountBlock.Data)
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err := contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || old.Timestamp == 0 {
		return ErrInvalidData
	}
	registerInfo, _ := contracts.ABI_register.PackVariable(contracts.VariableNameRegistration, old.Name, param.NodeAddr, old.PledgeAddr, param.BeneficialAddr, old.Amount, old.Timestamp, old.RewardHeight, old.CancelHeight)
	block.VmContext.SetStorage(key, registerInfo)
	return nil
}

type pVote struct {
}

func (p *pVote) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// vote for a super node of a consensus group
func (p *pVote) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, voteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamVote)
	err = contracts.ABI_vote.UnpackMethod(param, contracts.MethodNameVote, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}

	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.VoteConditionId, VoteConditionPrefix); !ok {
		return quotaLeft, ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.VoteConditionParam, block, param, contracts.MethodNameVote) {
		return quotaLeft, ErrInvalidData
	}

	return quotaLeft, nil
}

func (p *pVote) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamVote)
	contracts.ABI_vote.UnpackMethod(param, contracts.MethodNameVote, block.AccountBlock.Data)
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := contracts.GetVoteKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := contracts.ABI_vote.PackVariable(contracts.VariableNameVoteStatus, param.NodeName)
	block.VmContext.SetStorage(locHash, voteStatus)
	return nil
}

type pCancelVote struct {
}

func (p *pCancelVote) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel vote for a super node of a consensus group
func (p *pCancelVote) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	err = contracts.ABI_vote.UnpackMethod(gid, contracts.MethodNameCancelVote, block.AccountBlock.Data)
	if err != nil || !isExistGid(block.VmContext, *gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pCancelVote) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	contracts.ABI_vote.UnpackMethod(gid, contracts.MethodNameCancelVote, block.AccountBlock.Data)
	locHash := contracts.GetVoteKey(sendBlock.AccountAddress, *gid)
	block.VmContext.SetStorage(locHash, nil)
	return nil
}

type pPledge struct{}

func (p *pPledge) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// pledge ViteToken for a beneficial to get quota
func (p *pPledge) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, pledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() == 0 ||
		!IsViteToken(block.AccountBlock.TokenId) ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ParamPledge)
	err = contracts.ABI_pledge.UnpackMethod(param, contracts.MethodNamePledge, block.AccountBlock.Data)
	if err != nil || !block.VmContext.IsAddressExisted(&param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}

	if time.Unix(param.WithdrawTime-pledgeTime, 0).Before(*block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash).Timestamp) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pPledge) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamPledge)
	contracts.ABI_pledge.UnpackMethod(param, contracts.MethodNamePledge, block.AccountBlock.Data)
	// storage key for pledge beneficial: hash(beneficial)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes()).Bytes()
	// storage key for pledge: hash(owner, hash(beneficial))
	locHashPledge := types.DataHash(append(sendBlock.AccountAddress.Bytes(), locHashBeneficial...)).Bytes()
	oldPledgeData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, locHashPledge)
	amount := new(big.Int)
	if len(oldPledgeData) > 0 {
		oldPledge := new(contracts.VariablePledgeInfo)
		contracts.ABI_pledge.UnpackVariable(oldPledge, contracts.VariableNamePledgeInfo, oldPledgeData)
		if param.WithdrawTime < oldPledge.WithdrawTime {
			return ErrInvalidData
		}
		amount = oldPledge.Amount
	}
	amount.Add(amount, block.AccountBlock.Amount)
	pledgeInfo, _ := contracts.ABI_pledge.PackVariable(contracts.VariableNamePledgeInfo, amount, param.WithdrawTime)
	block.VmContext.SetStorage(locHashPledge, pledgeInfo)

	// storage value for quota: quota amount(0:32)
	oldBeneficialData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, locHashBeneficial)
	beneficialAmount := new(big.Int)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(contracts.VariablePledgeBeneficial)
		contracts.ABI_pledge.UnpackVariable(oldBeneficial, contracts.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, block.AccountBlock.Amount)
	beneficialData, _ := contracts.ABI_pledge.PackVariable(contracts.VariableNamePledgeBeneficial, beneficialAmount)
	block.VmContext.SetStorage(locHashBeneficial, beneficialData)
	return nil
}

type pCancelPledge struct{}

func (p *pCancelPledge) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel pledge ViteToken
func (p *pCancelPledge) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() > 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ParamCancelPledge)
	err = contracts.ABI_pledge.UnpackMethod(param, contracts.MethodNameCancelPledge, block.AccountBlock.Data)
	if err != nil || !block.VmContext.IsAddressExisted(&param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pCancelPledge) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamCancelPledge)
	contracts.ABI_pledge.UnpackMethod(param, contracts.MethodNameCancelPledge, block.AccountBlock.Data)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes()).Bytes()
	locHashPledge := types.DataHash(append(sendBlock.AccountAddress.Bytes(), locHashBeneficial...)).Bytes()
	oldPledge := new(contracts.VariablePledgeInfo)
	err := contracts.ABI_pledge.UnpackVariable(oldPledge, contracts.VariableNamePledgeInfo, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, locHashPledge))
	if err != nil || time.Unix(oldPledge.WithdrawTime, 0).After(*block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash).Timestamp) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(contracts.VariablePledgeBeneficial)
	err = contracts.ABI_pledge.UnpackVariable(oldBeneficial, contracts.VariableNamePledgeBeneficial, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, locHashBeneficial))
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		block.VmContext.SetStorage(locHashPledge, nil)
	} else {
		pledgeInfo, _ := contracts.ABI_pledge.PackVariable(contracts.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawTime)
		block.VmContext.SetStorage(locHashPledge, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		block.VmContext.SetStorage(locHashBeneficial, nil)
	} else {
		pledgeBeneficial, _ := contracts.ABI_pledge.PackVariable(contracts.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		block.VmContext.SetStorage(locHashBeneficial, pledgeBeneficial)
	}

	// append refund block
	vm.blockList = append(vm.blockList, &vm_context.VmAccountBlock{makeSendBlock(block.AccountBlock, sendBlock.AccountAddress, ledger.BlockTypeSendCall, param.Amount, *ledger.ViteTokenId(), []byte{}), nil})
	return nil
}

type pCreateConsensusGroup struct{}

func (p *pCreateConsensusGroup) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return new(big.Int).Set(createConsensusGroupFee), nil
}

// create consensus group
func (p *pCreateConsensusGroup) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, createConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ConsensusGroupInfo)
	err = contracts.ABI_consensusGroup.UnpackMethod(param, contracts.MethodNameCreateConsensusGroup, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := p.checkCreateConsensusGroupData(block.VmContext, param); err != nil {
		return quotaLeft, err
	}
	// data: methodSelector(0:4) + gid(4:36) + ConsensusGroup
	gid := types.DataToGid(block.AccountBlock.AccountAddress.Bytes(), new(big.Int).SetUint64(block.AccountBlock.Height).Bytes(), block.AccountBlock.PrevHash.Bytes(), block.AccountBlock.SnapshotHash.Bytes())
	if isExistGid(block.VmContext, gid) {
		return quotaLeft, ErrInvalidData
	}
	paramData, _ := contracts.ABI_consensusGroup.PackMethod(contracts.MethodNameCreateConsensusGroup, gid, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	block.AccountBlock.Data = paramData
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func (p *pCreateConsensusGroup) checkCreateConsensusGroupData(db vmctxt_interface.VmDatabase, param *contracts.ConsensusGroupInfo) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax {
		return ErrInvalidData
	}
	if err := p.checkCondition(db, param.CountingRuleId, param.CountingRuleParam, CountingRulePrefix); err != nil {
		return ErrInvalidData
	}
	if err := p.checkCondition(db, param.RegisterConditionId, param.RegisterConditionParam, RegisterConditionPrefix); err != nil {
		return ErrInvalidData
	}
	if err := p.checkCondition(db, param.VoteConditionId, param.VoteConditionParam, VoteConditionPrefix); err != nil {
		return ErrInvalidData
	}
	return nil
}
func (p *pCreateConsensusGroup) checkCondition(db vmctxt_interface.VmDatabase, conditionId uint8, conditionParam []byte, conditionIdPrefix uint) error {
	condition, ok := getConsensusGroupCondition(conditionId, conditionIdPrefix)
	if !ok {
		return ErrInvalidData
	}
	if ok := condition.checkParam(conditionParam, db); !ok {
		return ErrInvalidData
	}
	return nil
}
func (p *pCreateConsensusGroup) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ConsensusGroupInfo)
	contracts.ABI_consensusGroup.UnpackMethod(param, contracts.MethodNameCreateConsensusGroup, block.AccountBlock.Data)
	key := contracts.GetConsensusGroupKey(param.Gid)
	if len(block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)) > 0 {
		return ErrIdCollision
	}
	groupInfo, _ := contracts.ABI_consensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	block.VmContext.SetStorage(key, groupInfo)
	return nil
}

type CountingRuleCode uint

const (
	CountingRulePrefix                           = 0
	RegisterConditionPrefix                      = 10
	VoteConditionPrefix                          = 20
	CountingRuleOfBalance       CountingRuleCode = 0
	RegisterConditionOfSnapshot CountingRuleCode = 10
	VoteConditionOfDefault      CountingRuleCode = 20
	VoteConditionOfBalance      CountingRuleCode = 21
)

type createConsensusGroupCondition interface {
	checkParam(param []byte, db vmctxt_interface.VmDatabase) bool
	checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool
}

var SimpleCountingRuleList = map[CountingRuleCode]createConsensusGroupCondition{
	CountingRuleOfBalance:       &countingRuleOfBalance{},
	RegisterConditionOfSnapshot: &registerConditionOfPledge{},
	VoteConditionOfDefault:      &voteConditionOfDefault{},
	VoteConditionOfBalance:      &voteConditionOfKeepToken{},
}

func getConsensusGroupCondition(conditionId uint8, conditionIdPrefix uint) (createConsensusGroupCondition, bool) {
	condition, ok := SimpleCountingRuleList[CountingRuleCode(conditionIdPrefix+uint(conditionId))]
	return condition, ok
}

type countingRuleOfBalance struct{}

func (c countingRuleOfBalance) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(types.TokenTypeId)
	err := contracts.ABI_consensusGroup.UnpackVariable(v, contracts.VariableNameConditionCountingOfBalance, param)
	if err != nil || contracts.GetTokenById(db, *v) == nil {
		return false
	}
	return true
}
func (c countingRuleOfBalance) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	return true
}

type registerConditionOfPledge struct{}

func (c registerConditionOfPledge) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(contracts.VariableConditionRegisterOfPledge)
	err := contracts.ABI_consensusGroup.UnpackVariable(v, contracts.VariableNameConditionRegisterOfPledge, param)
	if err != nil || contracts.GetTokenById(db, v.PledgeToken) == nil {
		return false
	}
	return true
}

func (c registerConditionOfPledge) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	switch method {
	case contracts.MethodNameRegister:
		blockParam := blockParamInterface.(*contracts.ParamRegister)
		if !block.VmContext.IsAddressExisted(&blockParam.BeneficialAddr) ||
			!block.VmContext.IsAddressExisted(&blockParam.NodeAddr) ||
			!isUserAccount(block.VmContext, blockParam.NodeAddr) {
			return false
		}
		param := new(contracts.VariableConditionRegisterOfPledge)
		contracts.ABI_consensusGroup.UnpackVariable(param, contracts.VariableNameConditionRegisterOfPledge, paramData)
		if block.AccountBlock.Amount.Cmp(param.PledgeAmount) != 0 || !bytes.Equal(block.AccountBlock.TokenId.Bytes(), param.PledgeToken.Bytes()) {
			return false
		}
	case contracts.MethodNameCancelRegister:
		if block.AccountBlock.Amount.Sign() != 0 ||
			!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
			return false
		}

		param := new(contracts.VariableConditionRegisterOfPledge)
		contracts.ABI_consensusGroup.UnpackVariable(param, contracts.VariableNameConditionRegisterOfPledge, paramData)

		blockParam := blockParamInterface.(*contracts.ParamCancelRegister)
		key := contracts.GetRegisterKey(blockParam.Name, blockParam.Gid)
		old := new(contracts.Registration)
		err := contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.ToAddress, key))
		if err != nil || !bytes.Equal(old.PledgeAddr.Bytes(), block.AccountBlock.AccountAddress.Bytes()) ||
			old.Timestamp == 0 ||
			old.Timestamp+param.PledgeTime < block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash).Timestamp.Unix() {
			return false
		}
	case contracts.MethodNameUpdateRegistration:
		if block.AccountBlock.Amount.Sign() != 0 {
			return false
		}
		blockParam := blockParamInterface.(*contracts.ParamRegister)
		if !block.VmContext.IsAddressExisted(&blockParam.BeneficialAddr) ||
			!block.VmContext.IsAddressExisted(&blockParam.NodeAddr) ||
			!isUserAccount(block.VmContext, blockParam.NodeAddr) {
			return false
		}
		old := new(contracts.Registration)
		err := contracts.ABI_register.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&contracts.AddressRegister, contracts.GetRegisterKey(blockParam.Name, blockParam.Gid)))
		if err != nil ||
			!bytes.Equal(old.PledgeAddr.Bytes(), block.AccountBlock.AccountAddress.Bytes()) ||
			old.Timestamp == 0 ||
			(bytes.Equal(old.BeneficialAddr.Bytes(), blockParam.BeneficialAddr.Bytes()) && bytes.Equal(old.NodeAddr.Bytes(), blockParam.BeneficialAddr.Bytes())) {
			return false
		}
	}
	return true
}

type voteConditionOfDefault struct{}

func (c voteConditionOfDefault) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	if len(param) != 0 {
		return false
	}
	return true
}
func (c voteConditionOfDefault) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return false
	}
	return true
}

type voteConditionOfKeepToken struct{}

func (c voteConditionOfKeepToken) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(contracts.VariableConditionVoteOfKeepToken)
	err := contracts.ABI_consensusGroup.UnpackVariable(v, contracts.VariableNameConditionVoteOfKeepToken, param)
	if err != nil || contracts.GetTokenById(db, v.KeepToken) == nil {
		return false
	}
	return true
}
func (c voteConditionOfKeepToken) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return false
	}
	param := new(contracts.VariableConditionVoteOfKeepToken)
	contracts.ABI_consensusGroup.UnpackVariable(param, contracts.VariableNameConditionVoteOfKeepToken, paramData)
	if block.VmContext.GetBalance(&block.AccountBlock.AccountAddress, &param.KeepToken).Cmp(param.KeepAmount) < 0 {
		return false
	}
	return true
}

type pMintage struct{}

func (p *pMintage) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	if block.AccountBlock.Amount.Cmp(mintagePledgeAmount) == 0 {
		return big.NewInt(0), nil
	} else if block.AccountBlock.Amount.Sign() > 0 {
		return big.NewInt(0), ErrInvalidData
	}
	return new(big.Int).Set(mintageFee), nil
}

func (p *pMintage) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, mintageGas)
	if err != nil {
		return quotaLeft, err
	}
	param := new(contracts.ParamMintage)
	err = contracts.ABI_mintage.UnpackMethod(param, contracts.MethodNameMintage, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err = checkToken(*param); err != nil {
		return quotaLeft, err
	}
	tokenId := types.CreateTokenTypeId(block.AccountBlock.AccountAddress.Bytes(), new(big.Int).SetUint64(block.AccountBlock.Height).Bytes(), block.AccountBlock.PrevHash.Bytes(), block.AccountBlock.SnapshotHash.Bytes())
	if contracts.GetTokenById(block.VmContext, tokenId) != nil {
		return quotaLeft, ErrIdCollision
	}
	block.AccountBlock.Data, _ = contracts.ABI_mintage.PackMethod(contracts.MethodNameMintage, tokenId, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals)
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func checkToken(param contracts.ParamMintage) error {
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
func (p *pMintage) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamMintage)
	contracts.ABI_mintage.UnpackMethod(param, contracts.MethodNameMintage, block.AccountBlock.Data)
	key := helper.LeftPadBytes(param.TokenId.Bytes(), 32)
	if len(block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)) > 0 {
		return ErrIdCollision
	}
	var tokenInfo []byte
	if block.AccountBlock.Amount.Sign() == 0 {
		tokenInfo, _ = contracts.ABI_mintage.PackVariable(contracts.VariableNameMintage, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals, sendBlock.AccountAddress, sendBlock.Amount, int64(0))
	} else {
		tokenInfo, _ = contracts.ABI_mintage.PackVariable(contracts.VariableNameMintage, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals, sendBlock.AccountAddress, sendBlock.Amount, block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash).Timestamp.Unix()+mintagePledgeTime)
	}
	block.VmContext.SetStorage(key, tokenInfo)
	vm.blockList = append(vm.blockList, &vm_context.VmAccountBlock{makeSendBlock(block.AccountBlock, sendBlock.AccountAddress, ledger.BlockTypeSendReward, param.TotalSupply, param.TokenId, []byte{}), nil})
	return nil
}

type pMintageCancelPledge struct{}

func (p *pMintageCancelPledge) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *pMintageCancelPledge) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, mintageCancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() > 0 {
		return quotaLeft, ErrInvalidData
	}
	tokenId := new(types.TokenTypeId)
	err = contracts.ABI_mintage.UnpackMethod(tokenId, contracts.MethodNameMintageCancelPledge, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}
	tokenInfo := contracts.GetTokenById(block.VmContext, *tokenId)
	if !bytes.Equal(tokenInfo.Owner.Bytes(), block.AccountBlock.AccountAddress.Bytes()) || tokenInfo.PledgeAmount.Sign() == 0 || tokenInfo.Timestamp > block.VmContext.GetSnapshotBlock(&block.AccountBlock.SnapshotHash).Timestamp.Unix() {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pMintageCancelPledge) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	tokenId := new(types.TokenTypeId)
	contracts.ABI_mintage.UnpackMethod(tokenId, contracts.MethodNameMintageCancelPledge, block.AccountBlock.Data)
	storageKey := helper.LeftPadBytes(tokenId.Bytes(), types.HashSize)
	tokenInfo := new(contracts.TokenInfo)
	contracts.ABI_mintage.UnpackVariable(tokenInfo, contracts.VariableNameMintage, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, storageKey))
	newTokenInfo, _ := contracts.ABI_mintage.PackVariable(contracts.VariableNameMintage, tokenInfo.TokenName, tokenInfo.TokenSymbol, tokenInfo.TotalSupply, tokenInfo.Decimals, tokenInfo.Owner, big.NewInt(0), int64(0))
	block.VmContext.SetStorage(storageKey, newTokenInfo)
	if tokenInfo.PledgeAmount.Sign() > 0 {
		vm.blockList = append(vm.blockList, &vm_context.VmAccountBlock{makeSendBlock(block.AccountBlock, tokenInfo.Owner, ledger.BlockTypeSendCall, tokenInfo.PledgeAmount, *ledger.ViteTokenId(), []byte{}), nil})
	}
	return nil
}
func isUserAccount(db vmctxt_interface.VmDatabase, addr types.Address) bool {
	return len(db.GetContractCode(&addr)) == 0
}
