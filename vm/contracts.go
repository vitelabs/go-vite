package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"regexp"
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
		contracts.ABIRegister,
	},
	contracts.AddressVote: {
		map[string]precompiledContractMethod{
			contracts.MethodNameVote:       &pVote{},
			contracts.MethodNameCancelVote: &pCancelVote{},
		},
		contracts.ABIVote,
	},
	contracts.AddressPledge: {
		map[string]precompiledContractMethod{
			contracts.MethodNamePledge:       &pPledge{},
			contracts.MethodNameCancelPledge: &pCancelPledge{},
		},
		contracts.ABIPledge,
	},
	contracts.AddressConsensusGroup: {
		map[string]precompiledContractMethod{
			contracts.MethodNameCreateConsensusGroup:   &pCreateConsensusGroup{},
			contracts.MethodNameCancelConsensusGroup:   &pCancelConsensusGroup{},
			contracts.MethodNameReCreateConsensusGroup: &pReCreateConsensusGroup{},
		},
		contracts.ABIConsensusGroup,
	},
	contracts.AddressMintage: {
		map[string]precompiledContractMethod{
			contracts.MethodNameMintage:             &pMintage{},
			contracts.MethodNameMintageCancelPledge: &pMintageCancelPledge{},
		},
		contracts.ABIMintage,
	},
}

func isPrecompiledContractAddress(addr types.Address) bool {
	_, ok := simpleContracts[addr]
	return ok
}
func getPrecompiledContract(addr types.Address, methodSelector []byte) (precompiledContractMethod, bool, error) {
	p, ok := simpleContracts[addr]
	if ok {
		if method, err := p.abi.MethodById(methodSelector); err == nil {
			c, ok := p.m[method.Name]
			return c, ok, nil
		} else {
			return nil, ok, err
		}
	}
	return nil, false, nil
}

type pRegister struct {
}

func (p *pRegister) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *pRegister) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamRegister)
	err = contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameRegister, block.AccountBlock.Data)
	if err != nil || param.Gid == types.DELEGATE_GID {
		return quotaLeft, ErrInvalidData
	}
	if err = checkRegisterData(contracts.MethodNameRegister, block, param); err != nil {
		return quotaLeft, err
	}

	oldData := block.VmContext.GetStorage(&contracts.AddressRegister, contracts.GetRegisterKey(param.Name, param.Gid))
	if len(oldData) > 0 {
		old := new(contracts.Registration)
		contracts.ABIRegister.UnpackVariable(old, contracts.VariableNameRegistration, oldData)
		if old.IsActive() {
			// duplicate register
			return quotaLeft, ErrInvalidData
		}
	}
	return quotaLeft, nil
}

func checkRegisterData(methodName string, block *vm_context.VmAccountBlock, param *contracts.ParamRegister) error {
	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, contracts.RegisterConditionPrefix); !ok {
		return ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, methodName) {
		return ErrInvalidData
	}

	if types.PubkeyToAddress(param.PublicKey) != param.NodeAddr {
		return ErrInvalidData
	}

	if verified, err := crypto.VerifySig(
		param.PublicKey,
		contracts.GetRegisterMessageForSignature(block.AccountBlock.AccountAddress, block.AccountBlock.Height, block.AccountBlock.PrevHash, block.AccountBlock.SnapshotHash),
		param.Signature); !verified {
		return err
	}
	return nil
}

func (p *pRegister) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamRegister)
	contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameRegister, sendBlock.Data)
	snapshotBlock := block.VmContext.CurrentSnapshotBlock()
	rewardHeight := snapshotBlock.Height
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	oldData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)
	if len(oldData) > 0 {
		old := new(contracts.Registration)
		contracts.ABIRegister.UnpackVariable(old, contracts.VariableNameRegistration, oldData)
		if old.IsActive() {
			// duplicate register
			return ErrInvalidData
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
	}
	registerInfo, _ := contracts.ABIRegister.PackVariable(
		contracts.VariableNameRegistration,
		param.Name,
		param.NodeAddr,
		sendBlock.AccountAddress,
		param.BeneficialAddr,
		sendBlock.Amount,
		snapshotBlock.Height,
		rewardHeight,
		uint64(0))
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
	quotaLeft, err := quota.UseQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamCancelRegister)
	err = contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameCancelRegister, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}

	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, contracts.RegisterConditionPrefix); !ok {
		return quotaLeft, ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, contracts.MethodNameCancelRegister) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pCancelRegister) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamCancelRegister)
	contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameCancelRegister, sendBlock.Data)

	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err := contracts.ABIRegister.UnpackVariable(
		old,
		contracts.VariableNameRegistration,
		block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || !old.IsActive() {
		return ErrInvalidData
	}

	// update lock amount and loc start height
	snapshotBlock := block.VmContext.CurrentSnapshotBlock()
	registerInfo, _ := contracts.ABIRegister.PackVariable(
		contracts.VariableNameRegistration,
		param.Name, old.NodeAddr,
		old.PledgeAddr,
		old.BeneficialAddr,
		helper.Big0,
		uint64(0),
		old.RewardHeight,
		snapshotBlock.Height)
	block.VmContext.SetStorage(key, registerInfo)
	// return locked ViteToken
	if old.Amount.Sign() > 0 {
		vm.blockList = append(vm.blockList,
			&vm_context.VmAccountBlock{
				makeSendBlock(
					block.AccountBlock,
					sendBlock.AccountAddress,
					ledger.BlockTypeSendCall,
					old.Amount,
					ledger.ViteTokenId,
					vm.getNewBlockHeight(block),
					[]byte{}),
				nil})
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
	quotaLeft, err := quota.UseQuota(quotaLeft, rewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ParamReward)
	err = contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameReward, block.AccountBlock.Data)
	if err != nil || !IsSnapshotGid(param.Gid) {
		return quotaLeft, ErrInvalidData
	}
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err = contracts.ABIRegister.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.ToAddress, key))
	if err != nil || block.AccountBlock.AccountAddress != old.PledgeAddr {
		return quotaLeft, ErrInvalidData
	}
	if block.VmContext.CurrentSnapshotBlock().Height < rewardHeightLimit {
		return quotaLeft, ErrInvalidData
	}

	if param.EndHeight == 0 {
		param.EndHeight = block.VmContext.CurrentSnapshotBlock().Height - rewardHeightLimit
		if !old.IsActive() {
			param.EndHeight = helper.Min(param.EndHeight, old.CancelHeight)
		}
	} else if param.EndHeight > block.VmContext.CurrentSnapshotBlock().Height-rewardHeightLimit ||
		param.EndHeight > old.CancelHeight {
		return quotaLeft, ErrInvalidData
	}

	if param.StartHeight == 0 {
		param.StartHeight = old.RewardHeight
	}

	if param.EndHeight <= param.StartHeight {
		return quotaLeft, ErrInvalidData
	}

	count := param.EndHeight - param.StartHeight
	// avoid uint64 overflow
	if count > maxRewardCount {
		return quotaLeft, err
	}
	if quotaLeft, err = quota.UseQuota(quotaLeft, ((count+dbPageSize-1)/dbPageSize)*calcRewardGasPerPage); err != nil {
		return quotaLeft, err
	}

	// calc snapshot block produce reward between param.StartHeight(excluded) and param.EndHeight(included)
	calcReward(block.VmContext, old.NodeAddr, param.StartHeight, count, param.Amount)
	block.AccountBlock.Data, err = contracts.ABIRegister.PackMethod(
		contracts.MethodNameReward,
		param.Gid,
		param.Name,
		param.EndHeight,
		param.StartHeight,
		param.Amount)
	if err != nil {
		return quotaLeft, err
	}
	if quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func calcReward(db vmctxt_interface.VmDatabase, producer types.Address, startHeight uint64, count uint64, reward *big.Int) {
	startHeight = startHeight + 1
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
			if block.Producer() == producer {
				rewardCount++
			}
		}
	}
	reward.SetUint64(rewardCount)
	reward.Mul(rewardPerBlock, reward)
}
func (p *pReward) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamReward)
	contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameReward, sendBlock.Data)
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err := contracts.ABIRegister.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || old.RewardHeight > param.StartHeight || sendBlock.AccountAddress != old.PledgeAddr {
		return ErrInvalidData
	}
	if !old.IsActive() {
		if param.EndHeight > old.CancelHeight {
			return ErrInvalidData
		} else {
			// get reward partly, update storage
			registerInfo, _ := contracts.ABIRegister.PackVariable(
				contracts.VariableNameRegistration,
				old.Name,
				old.NodeAddr,
				old.PledgeAddr,
				old.BeneficialAddr,
				old.Amount,
				old.PledgeHeight,
				param.EndHeight,
				old.CancelHeight)
			block.VmContext.SetStorage(key, registerInfo)
		}
	} else {
		registerInfo, _ := contracts.ABIRegister.PackVariable(
			contracts.VariableNameRegistration,
			old.Name,
			old.NodeAddr,
			old.PledgeAddr,
			old.BeneficialAddr,
			old.Amount,
			old.PledgeHeight,
			param.EndHeight,
			old.CancelHeight)
		block.VmContext.SetStorage(key, registerInfo)
	}

	if param.Amount.Sign() > 0 {
		// create reward and return
		vm.blockList = append(vm.blockList,
			&vm_context.VmAccountBlock{
				makeSendBlock(
					block.AccountBlock,
					old.BeneficialAddr,
					ledger.BlockTypeSendReward,
					param.Amount,
					ledger.ViteTokenId,
					vm.getNewBlockHeight(block),
					[]byte{}),
				nil})
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
	quotaLeft, err := quota.UseQuota(quotaLeft, updateRegistrationGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamRegister)
	err = contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameUpdateRegistration, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}

	if err = checkRegisterData(contracts.MethodNameUpdateRegistration, block, param); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func (p *pUpdateRegistration) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamRegister)
	contracts.ABIRegister.UnpackMethod(param, contracts.MethodNameUpdateRegistration, sendBlock.Data)
	key := contracts.GetRegisterKey(param.Name, param.Gid)
	old := new(contracts.Registration)
	err := contracts.ABIRegister.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || !old.IsActive() {
		return ErrInvalidData
	}
	registerInfo, _ := contracts.ABIRegister.PackVariable(
		contracts.VariableNameRegistration,
		old.Name, param.NodeAddr,
		old.PledgeAddr,
		param.BeneficialAddr,
		old.Amount,
		old.PledgeHeight,
		old.RewardHeight,
		old.CancelHeight)
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
	quotaLeft, err := quota.UseQuota(quotaLeft, voteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}

	param := new(contracts.ParamVote)
	err = contracts.ABIVote.UnpackMethod(param, contracts.MethodNameVote, block.AccountBlock.Data)
	if err != nil || param.Gid == types.DELEGATE_GID {
		return quotaLeft, ErrInvalidData
	}

	consensusGroupInfo := contracts.GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, ErrInvalidData
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.VoteConditionId, contracts.VoteConditionPrefix); !ok {
		return quotaLeft, ErrInvalidData
	} else if !condition.checkData(consensusGroupInfo.VoteConditionParam, block, param, contracts.MethodNameVote) {
		return quotaLeft, ErrInvalidData
	}

	return quotaLeft, nil
}

func (p *pVote) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamVote)
	contracts.ABIVote.UnpackMethod(param, contracts.MethodNameVote, sendBlock.Data)
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := contracts.GetVoteKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := contracts.ABIVote.PackVariable(contracts.VariableNameVoteStatus, param.NodeName)
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
	quotaLeft, err := quota.UseQuota(quotaLeft, cancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	err = contracts.ABIVote.UnpackMethod(gid, contracts.MethodNameCancelVote, block.AccountBlock.Data)
	if err != nil || !isExistGid(block.VmContext, *gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pCancelVote) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	contracts.ABIVote.UnpackMethod(gid, contracts.MethodNameCancelVote, sendBlock.Data)
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
	// pledge gas is low without data gas cost, so that a new account is easy to pledge
	quotaLeft, err := quota.UseQuota(quotaLeft, pledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Cmp(pledgeAmountMin) < 0 ||
		!IsViteToken(block.AccountBlock.TokenId) ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	beneficialAddr := new(types.Address)
	err = contracts.ABIPledge.UnpackMethod(beneficialAddr, contracts.MethodNamePledge, block.AccountBlock.Data)
	if err != nil || !block.VmContext.IsAddressExisted(beneficialAddr) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pPledge) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	beneficialAddr := new(types.Address)
	contracts.ABIPledge.UnpackMethod(beneficialAddr, contracts.MethodNamePledge, sendBlock.Data)
	beneficialKey := contracts.GetPledgeBeneficialKey(*beneficialAddr)
	pledgeKey := contracts.GetPledgeKey(sendBlock.AccountAddress, beneficialKey)
	oldPledgeData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, pledgeKey)
	amount := new(big.Int)
	if len(oldPledgeData) > 0 {
		oldPledge := new(contracts.PledgeInfo)
		contracts.ABIPledge.UnpackVariable(oldPledge, contracts.VariableNamePledgeInfo, oldPledgeData)
		amount = oldPledge.Amount
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeInfo, amount, block.VmContext.CurrentSnapshotBlock().Height+minPledgeHeight)
	block.VmContext.SetStorage(pledgeKey, pledgeInfo)

	oldBeneficialData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, beneficialKey)
	beneficialAmount := new(big.Int)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(contracts.VariablePledgeBeneficial)
		contracts.ABIPledge.UnpackVariable(oldBeneficial, contracts.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, beneficialAmount)
	block.VmContext.SetStorage(beneficialKey, beneficialData)
	return nil
}

type pCancelPledge struct{}

func (p *pCancelPledge) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel pledge ViteToken
func (p *pCancelPledge) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, cancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() > 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ParamCancelPledge)
	err = contracts.ABIPledge.UnpackMethod(param, contracts.MethodNameCancelPledge, block.AccountBlock.Data)
	if err != nil || !block.VmContext.IsAddressExisted(&param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pCancelPledge) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(contracts.ParamCancelPledge)
	contracts.ABIPledge.UnpackMethod(param, contracts.MethodNameCancelPledge, sendBlock.Data)
	beneficialKey := contracts.GetPledgeBeneficialKey(param.Beneficial)
	pledgeKey := contracts.GetPledgeKey(sendBlock.AccountAddress, beneficialKey)
	oldPledge := new(contracts.PledgeInfo)
	err := contracts.ABIPledge.UnpackVariable(oldPledge, contracts.VariableNamePledgeInfo, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, pledgeKey))
	if err != nil || oldPledge.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(contracts.VariablePledgeBeneficial)
	err = contracts.ABIPledge.UnpackVariable(oldBeneficial, contracts.VariableNamePledgeBeneficial, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, beneficialKey))
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		block.VmContext.SetStorage(pledgeKey, nil)
	} else {
		pledgeInfo, _ := contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight)
		block.VmContext.SetStorage(pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		block.VmContext.SetStorage(beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := contracts.ABIPledge.PackVariable(contracts.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		block.VmContext.SetStorage(beneficialKey, pledgeBeneficial)
	}

	vm.blockList = append(vm.blockList,
		&vm_context.VmAccountBlock{
			makeSendBlock(
				block.AccountBlock,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				param.Amount,
				ledger.ViteTokenId,
				vm.getNewBlockHeight(block),
				[]byte{}),
			nil})
	return nil
}

type pCreateConsensusGroup struct{}

func (p *pCreateConsensusGroup) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *pCreateConsensusGroup) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, createConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!IsViteToken(block.AccountBlock.TokenId) ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(contracts.ConsensusGroupInfo)
	err = contracts.ABIConsensusGroup.UnpackMethod(param, contracts.MethodNameCreateConsensusGroup, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := checkCreateConsensusGroupData(block.VmContext, param); err != nil {
		return quotaLeft, err
	}
	gid := contracts.NewGid(block.AccountBlock.AccountAddress, block.AccountBlock.Height, block.AccountBlock.PrevHash, block.AccountBlock.SnapshotHash)
	if isExistGid(block.VmContext, gid) {
		return quotaLeft, ErrInvalidData
	}
	paramData, _ := contracts.ABIConsensusGroup.PackMethod(
		contracts.MethodNameCreateConsensusGroup,
		gid,
		param.NodeCount,
		param.Interval,
		param.PerCount,
		param.RandCount,
		param.RandRank,
		param.CountingTokenId,
		param.RegisterConditionId,
		param.RegisterConditionParam,
		param.VoteConditionId,
		param.VoteConditionParam)
	block.AccountBlock.Data = paramData
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func checkCreateConsensusGroupData(db vmctxt_interface.VmDatabase, param *contracts.ConsensusGroupInfo) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax ||
		param.PerCount < cgPerCountMin || param.PerCount > cgPerCountMax ||
		// no overflow
		param.PerCount*param.Interval < cgPerIntervalMin || param.PerCount*param.Interval > cgPerIntervalMax ||
		param.RandCount > param.NodeCount ||
		(param.RandCount > 0 && param.RandRank < param.NodeCount) {
		return ErrInvalidData
	}
	if contracts.GetTokenById(db, param.CountingTokenId) == nil {
		return ErrInvalidData
	}
	if err := checkCondition(db, param.RegisterConditionId, param.RegisterConditionParam, contracts.RegisterConditionPrefix); err != nil {
		return ErrInvalidData
	}
	if err := checkCondition(db, param.VoteConditionId, param.VoteConditionParam, contracts.VoteConditionPrefix); err != nil {
		return ErrInvalidData
	}
	return nil
}
func checkCondition(db vmctxt_interface.VmDatabase, conditionId uint8, conditionParam []byte, conditionIdPrefix contracts.ConditionCode) error {
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
	contracts.ABIConsensusGroup.UnpackMethod(param, contracts.MethodNameCreateConsensusGroup, sendBlock.Data)
	key := contracts.GetConsensusGroupKey(param.Gid)
	if len(block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)) > 0 {
		return ErrIdCollision
	}
	groupInfo, _ := contracts.ABIConsensusGroup.PackVariable(
		contracts.VariableNameConsensusGroupInfo,
		param.NodeCount,
		param.Interval,
		param.PerCount,
		param.RandCount,
		param.RandRank,
		param.CountingTokenId,
		param.RegisterConditionId,
		param.RegisterConditionParam,
		param.VoteConditionId,
		param.VoteConditionParam,
		sendBlock.AccountAddress,
		sendBlock.Amount,
		block.VmContext.CurrentSnapshotBlock().Height+createConsensusGroupPledgeHeight)
	block.VmContext.SetStorage(key, groupInfo)
	return nil
}

type pCancelConsensusGroup struct{}

func (p *pCancelConsensusGroup) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// Cancel consensus group and get pledge back.
// A canceled consensus group(no-active) will not generate contract blocks after cancel receive block is confirmed.
// Consensus group name is kept even if canceled.
func (p *pCancelConsensusGroup) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, cancelConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	err = contracts.ABIConsensusGroup.UnpackMethod(gid, contracts.MethodNameCancelConsensusGroup, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	groupInfo := contracts.GetConsensusGroup(block.VmContext, *gid)
	if groupInfo == nil ||
		block.AccountBlock.AccountAddress != groupInfo.Owner ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pCancelConsensusGroup) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	contracts.ABIConsensusGroup.UnpackMethod(gid, contracts.MethodNameCancelConsensusGroup, sendBlock.Data)
	key := contracts.GetConsensusGroupKey(*gid)
	groupInfo := contracts.GetConsensusGroup(block.VmContext, *gid)
	if groupInfo == nil ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height {
		return ErrInvalidData
	}
	newGroupInfo, _ := contracts.ABIConsensusGroup.PackVariable(
		contracts.VariableNameConsensusGroupInfo,
		groupInfo.NodeCount,
		groupInfo.Interval,
		groupInfo.PerCount,
		groupInfo.RandCount,
		groupInfo.RandRank,
		groupInfo.CountingTokenId,
		groupInfo.RegisterConditionId,
		groupInfo.RegisterConditionParam,
		groupInfo.VoteConditionId,
		groupInfo.VoteConditionParam,
		groupInfo.Owner,
		helper.Big0,
		uint64(0))
	block.VmContext.SetStorage(key, newGroupInfo)
	if groupInfo.PledgeAmount.Sign() > 0 {
		vm.blockList = append(vm.blockList,
			&vm_context.VmAccountBlock{
				makeSendBlock(
					block.AccountBlock,
					sendBlock.AccountAddress,
					ledger.BlockTypeSendCall,
					groupInfo.PledgeAmount,
					ledger.ViteTokenId,
					vm.getNewBlockHeight(block),
					[]byte{},
				),
				nil,
			})
	}
	return nil
}

type pReCreateConsensusGroup struct{}

func (p *pReCreateConsensusGroup) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// Pledge again for a canceled consensus group.
// A consensus group will start generate contract blocks after recreate receive block is confirmed.
func (p *pReCreateConsensusGroup) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, reCreateConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft); err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!IsViteToken(block.AccountBlock.TokenId) ||
		!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	if err = contracts.ABIConsensusGroup.UnpackMethod(gid, contracts.MethodNameReCreateConsensusGroup, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	if groupInfo := contracts.GetConsensusGroup(block.VmContext, *gid); groupInfo == nil ||
		block.AccountBlock.AccountAddress != groupInfo.Owner ||
		groupInfo.IsActive() {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pReCreateConsensusGroup) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	contracts.ABIConsensusGroup.UnpackMethod(gid, contracts.MethodNameReCreateConsensusGroup, sendBlock.Data)
	key := contracts.GetConsensusGroupKey(*gid)
	groupInfo := contracts.GetConsensusGroup(block.VmContext, *gid)
	if groupInfo == nil ||
		groupInfo.IsActive() {
		return ErrInvalidData
	}
	newGroupInfo, _ := contracts.ABIConsensusGroup.PackVariable(
		contracts.VariableNameConsensusGroupInfo,
		groupInfo.NodeCount,
		groupInfo.Interval,
		groupInfo.PerCount,
		groupInfo.RandCount,
		groupInfo.RandRank,
		groupInfo.CountingTokenId,
		groupInfo.RegisterConditionId,
		groupInfo.RegisterConditionParam,
		groupInfo.VoteConditionId,
		groupInfo.VoteConditionParam,
		groupInfo.Owner,
		sendBlock.Amount,
		block.VmContext.CurrentSnapshotBlock().Height+createConsensusGroupPledgeHeight)
	block.VmContext.SetStorage(key, newGroupInfo)
	return nil
}

type createConsensusGroupCondition interface {
	checkParam(param []byte, db vmctxt_interface.VmDatabase) bool
	checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool
}

var SimpleCountingRuleList = map[contracts.ConditionCode]createConsensusGroupCondition{
	contracts.RegisterConditionOfPledge: &registerConditionOfPledge{},
	contracts.VoteConditionOfDefault:    &voteConditionOfDefault{},
	contracts.VoteConditionOfBalance:    &voteConditionOfKeepToken{},
}

func getConsensusGroupCondition(conditionId uint8, conditionIdPrefix contracts.ConditionCode) (createConsensusGroupCondition, bool) {
	condition, ok := SimpleCountingRuleList[conditionIdPrefix+contracts.ConditionCode(conditionId)]
	return condition, ok
}

type registerConditionOfPledge struct{}

func (c registerConditionOfPledge) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(contracts.VariableConditionRegisterOfPledge)
	err := contracts.ABIConsensusGroup.UnpackVariable(v, contracts.VariableNameConditionRegisterOfPledge, param)
	if err != nil ||
		contracts.GetTokenById(db, v.PledgeToken) == nil ||
		v.PledgeAmount.Sign() == 0 ||
		v.PledgeHeight < minPledgeHeight {
		return false
	}
	return true
}

func (c registerConditionOfPledge) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	switch method {
	case contracts.MethodNameRegister:
		blockParam := blockParamInterface.(*contracts.ParamRegister)
		if (blockParam.Gid == types.SNAPSHOT_GID && !block.VmContext.IsAddressExisted(&blockParam.BeneficialAddr)) ||
			!block.VmContext.IsAddressExisted(&blockParam.NodeAddr) ||
			!isUserAccount(block.VmContext, blockParam.NodeAddr) {
			return false
		}
		if ok, _ := regexp.MatchString("^[0-9a-zA-Z_.]{1,40}$", blockParam.Name); !ok {
			return false
		}
		param := new(contracts.VariableConditionRegisterOfPledge)
		contracts.ABIConsensusGroup.UnpackVariable(param, contracts.VariableNameConditionRegisterOfPledge, paramData)
		if block.AccountBlock.Amount.Cmp(param.PledgeAmount) != 0 || block.AccountBlock.TokenId != param.PledgeToken {
			return false
		}
	case contracts.MethodNameCancelRegister:
		if block.AccountBlock.Amount.Sign() != 0 ||
			!isUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
			return false
		}

		param := new(contracts.VariableConditionRegisterOfPledge)
		contracts.ABIConsensusGroup.UnpackVariable(param, contracts.VariableNameConditionRegisterOfPledge, paramData)

		blockParam := blockParamInterface.(*contracts.ParamCancelRegister)
		key := contracts.GetRegisterKey(blockParam.Name, blockParam.Gid)
		old := new(contracts.Registration)
		err := contracts.ABIRegister.UnpackVariable(old, contracts.VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.ToAddress, key))
		if err != nil || old.PledgeAddr != block.AccountBlock.AccountAddress ||
			!old.IsActive() ||
			old.PledgeHeight+param.PledgeHeight > block.VmContext.CurrentSnapshotBlock().Height {
			return false
		}
	case contracts.MethodNameUpdateRegistration:
		if block.AccountBlock.Amount.Sign() != 0 {
			return false
		}
		blockParam := blockParamInterface.(*contracts.ParamRegister)
		if (blockParam.Gid == types.SNAPSHOT_GID && !block.VmContext.IsAddressExisted(&blockParam.BeneficialAddr)) ||
			!block.VmContext.IsAddressExisted(&blockParam.NodeAddr) ||
			!isUserAccount(block.VmContext, blockParam.NodeAddr) {
			return false
		}
		old := new(contracts.Registration)
		err := contracts.ABIRegister.UnpackVariable(
			old,
			contracts.VariableNameRegistration,
			block.VmContext.GetStorage(&contracts.AddressRegister, contracts.GetRegisterKey(blockParam.Name, blockParam.Gid)))
		if err != nil ||
			old.PledgeAddr != block.AccountBlock.AccountAddress ||
			!old.IsActive() ||
			(old.BeneficialAddr == blockParam.BeneficialAddr && old.NodeAddr == blockParam.BeneficialAddr) {
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
	err := contracts.ABIConsensusGroup.UnpackVariable(v, contracts.VariableNameConditionVoteOfKeepToken, param)
	if err != nil || contracts.GetTokenById(db, v.KeepToken) == nil || v.KeepAmount.Sign() == 0 {
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
	contracts.ABIConsensusGroup.UnpackVariable(param, contracts.VariableNameConditionVoteOfKeepToken, paramData)
	if block.VmContext.GetBalance(&block.AccountBlock.AccountAddress, &param.KeepToken).Cmp(param.KeepAmount) < 0 {
		return false
	}
	return true
}

type pMintage struct{}

func (p *pMintage) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	if block.AccountBlock.Amount.Cmp(mintagePledgeAmount) == 0 && IsViteToken(block.AccountBlock.TokenId) {
		// Pledge ViteToken to mintage
		return big.NewInt(0), nil
	} else if block.AccountBlock.Amount.Sign() > 0 {
		return big.NewInt(0), ErrInvalidData
	}
	// Destroy ViteToken to mintage
	return new(big.Int).Set(mintageFee), nil
}

func (p *pMintage) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, mintageGas)
	if err != nil {
		return quotaLeft, err
	}
	param := new(contracts.ParamMintage)
	err = contracts.ABIMintage.UnpackMethod(param, contracts.MethodNameMintage, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err = checkToken(*param); err != nil {
		return quotaLeft, err
	}
	tokenId := contracts.NewTokenId(block.AccountBlock.AccountAddress, block.AccountBlock.Height, block.AccountBlock.PrevHash, block.AccountBlock.SnapshotHash)
	if contracts.GetTokenById(block.VmContext, tokenId) != nil {
		return quotaLeft, ErrIdCollision
	}
	block.AccountBlock.Data, _ = contracts.ABIMintage.PackMethod(
		contracts.MethodNameMintage,
		tokenId,
		param.TokenName,
		param.TokenSymbol,
		param.TotalSupply,
		param.Decimals)
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func checkToken(param contracts.ParamMintage) error {
	if param.TotalSupply.Cmp(helper.Tt256m1) > 0 ||
		param.TotalSupply.Cmp(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(param.Decimals)), nil)) < 0 ||
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
	contracts.ABIMintage.UnpackMethod(param, contracts.MethodNameMintage, sendBlock.Data)
	key := contracts.GetMintageKey(param.TokenId)
	if len(block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)) > 0 {
		return ErrIdCollision
	}
	var tokenInfo []byte
	if sendBlock.Amount.Sign() == 0 {
		tokenInfo, _ = contracts.ABIMintage.PackVariable(
			contracts.VariableNameMintage,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			uint64(0))
	} else {
		tokenInfo, _ = contracts.ABIMintage.PackVariable(
			contracts.VariableNameMintage,
			param.TokenName,
			param.TokenSymbol,
			param.TotalSupply,
			param.Decimals,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			block.VmContext.CurrentSnapshotBlock().Height+mintagePledgeHeight)
	}
	block.VmContext.SetStorage(key, tokenInfo)
	vm.blockList = append(vm.blockList,
		&vm_context.VmAccountBlock{
			makeSendBlock(
				block.AccountBlock,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendReward,
				param.TotalSupply,
				param.TokenId,
				vm.getNewBlockHeight(block),
				[]byte{}),
			nil})
	return nil
}

type pMintageCancelPledge struct{}

func (p *pMintageCancelPledge) getFee(vm *VM, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *pMintageCancelPledge) doSend(vm *VM, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := quota.UseQuota(quotaLeft, mintageCancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = quota.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() > 0 {
		return quotaLeft, ErrInvalidData
	}
	tokenId := new(types.TokenTypeId)
	err = contracts.ABIMintage.UnpackMethod(tokenId, contracts.MethodNameMintageCancelPledge, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, ErrInvalidData
	}
	tokenInfo := contracts.GetTokenById(block.VmContext, *tokenId)
	if tokenInfo.Owner != block.AccountBlock.AccountAddress ||
		tokenInfo.PledgeAmount.Sign() == 0 ||
		tokenInfo.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *pMintageCancelPledge) doReceive(vm *VM, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	tokenId := new(types.TokenTypeId)
	contracts.ABIMintage.UnpackMethod(tokenId, contracts.MethodNameMintageCancelPledge, sendBlock.Data)
	storageKey := contracts.GetMintageKey(*tokenId)
	tokenInfo := new(contracts.TokenInfo)
	contracts.ABIMintage.UnpackVariable(tokenInfo, contracts.VariableNameMintage, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, storageKey))
	newTokenInfo, _ := contracts.ABIMintage.PackVariable(
		contracts.VariableNameMintage,
		tokenInfo.TokenName,
		tokenInfo.TokenSymbol,
		tokenInfo.TotalSupply,
		tokenInfo.Decimals,
		tokenInfo.Owner,
		big.NewInt(0),
		int64(0))
	block.VmContext.SetStorage(storageKey, newTokenInfo)
	if tokenInfo.PledgeAmount.Sign() > 0 {
		vm.blockList = append(vm.blockList,
			&vm_context.VmAccountBlock{
				makeSendBlock(
					block.AccountBlock,
					tokenInfo.Owner,
					ledger.BlockTypeSendCall,
					tokenInfo.PledgeAmount,
					ledger.ViteTokenId,
					vm.getNewBlockHeight(block),
					[]byte{}),
				nil})
	}
	return nil
}
func isUserAccount(db vmctxt_interface.VmDatabase, addr types.Address) bool {
	return len(db.GetContractCode(&addr)) == 0
}
