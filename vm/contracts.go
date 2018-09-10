package vm

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strconv"
)

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressPledge, _         = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
)

type precompiledContract interface {
	createFee(vm *VM, block VmAccountBlock) *big.Int
	doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error)
	doReceive(vm *VM, block VmAccountBlock) error
}

var simpleContracts = map[types.Address]precompiledContract{
	AddressRegister:       &register{ABI_register},
	AddressVote:           &vote{ABI_vote},
	AddressPledge:         &pledge{ABI_pledge},
	AddressConsensusGroup: &consensusGroup{ABI_consensusGroup},
}

func getPrecompiledContract(address types.Address) (precompiledContract, bool) {
	p, ok := simpleContracts[address]
	return p, ok
}

type register struct {
	abi.ABIContract
}

func (p *register) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return big.NewInt(0)
}
func (p *register) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if method, err := p.MethodById(block.Data()[0:4]); err == nil {
		switch method.Name {
		case MethodNameRegister:
			return p.doSendRegister(vm, block, quotaLeft)
		case MethodNameCancelRegister:
			return p.doSendCancelRegister(vm, block, quotaLeft)
		case MethodNameReward:
			return p.doSendReward(vm, block, quotaLeft)
		}
	}
	return quotaLeft, ErrInvalidData
}

// register to become a super node of a consensus group, lock 100w ViteToken for 3 month
func (p *register) doSendRegister(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Cmp(registerAmount) != 0 ||
		!util.IsViteToken(block.TokenId()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}

	gid := new(types.Gid)
	err = p.UnpackMethod(gid, MethodNameRegister, block.Data())
	if err != nil || !isExistGid(vm.Db, *gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *register) doSendCancelRegister(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	err = p.UnpackMethod(gid, MethodNameCancelRegister, block.Data())
	if err != nil || !isExistGid(vm.Db, *gid) {
		return quotaLeft, ErrInvalidData
	}

	locHash := getKey(block.AccountAddress(), *gid)
	old := new(VariableRegistration)
	err = p.UnpackVariable(old, VariableNameRegistration, vm.Db.Storage(block.ToAddress(), locHash))
	if err != nil || old.Timestamp+registerLockTime < vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp() {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// get reward of generating snapshot block
func (p *register) doSendReward(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, rewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamReward)
	err = p.UnpackMethod(param, MethodNameReward, block.Data())
	if err != nil || !util.IsSnapshotGid(param.Gid) {
		return quotaLeft, ErrInvalidData
	}
	locHash := getKey(block.AccountAddress(), param.Gid)
	old := new(VariableRegistration)
	err = p.UnpackVariable(old, VariableNameRegistration, vm.Db.Storage(block.ToAddress(), locHash))
	if err != nil {
		return quotaLeft, ErrInvalidData
	}
	// newRewardHeight := min(currentSnapshotHeight-50, userDefined, cancelSnapshotHeight)
	newRewardHeight := new(big.Int).Sub(vm.Db.SnapshotBlock(block.SnapshotHash()).Height(), rewardHeightLimit)
	if param.EndHeight.Sign() > 0 {
		newRewardHeight = util.BigMin(newRewardHeight, param.EndHeight)
	}
	if old.CancelHeight.Sign() > 0 {
		newRewardHeight = util.BigMin(newRewardHeight, old.CancelHeight)
	}
	if newRewardHeight.Cmp(old.RewardHeight) <= 0 {
		return quotaLeft, ErrInvalidData
	}
	heightGap := new(big.Int).Sub(newRewardHeight, old.RewardHeight)
	if heightGap.Cmp(rewardGapLimit) > 0 {
		return quotaLeft, ErrInvalidData
	}

	count := heightGap.Uint64()
	quotaLeft, err = useQuota(quotaLeft, ((count+dbPageSize-1)/dbPageSize)*calcRewardGasPerPage)
	if err != nil {
		return quotaLeft, err
	}

	calcReward(vm, block.AccountAddress().Bytes(), old.RewardHeight, count, param.Amount)
	data, err := p.PackMethod(MethodNameReward, param.Gid, newRewardHeight, old.RewardHeight, param.Amount)
	if err != nil {
		return quotaLeft, err
	}
	block.SetData(data)
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

func calcReward(vm *VM, producer []byte, startHeight *big.Int, count uint64, reward *big.Int) {
	var rewardCount uint64
	for count > 0 {
		var list []VmSnapshotBlock
		if count < dbPageSize {
			list = vm.Db.SnapshotBlockList(startHeight, count, true)
			count = 0
		} else {
			list = vm.Db.SnapshotBlockList(startHeight, dbPageSize, true)
			count = count - dbPageSize
			startHeight.Add(startHeight, dbPageSizeBig)
		}
		for _, block := range list {
			if bytes.Equal(block.Producer().Bytes(), producer) {
				rewardCount++
			}
		}
	}
	reward.SetUint64(rewardCount)
	reward.Mul(rewardPerBlock, reward)
}

func (p *register) doReceive(vm *VM, block VmAccountBlock) error {
	if method, err := p.MethodById(block.Data()[0:4]); err == nil {
		switch method.Name {
		case MethodNameRegister:
			return p.doReceiveRegister(vm, block)
		case MethodNameCancelRegister:
			return p.doReceiveCancelRegister(vm, block)
		case MethodNameReward:
			return p.doReceiveReward(vm, block)
		}
	}
	return ErrInvalidData
}
func (p *register) doReceiveRegister(vm *VM, block VmAccountBlock) error {
	gid := new(types.Gid)
	p.UnpackMethod(gid, MethodNameRegister, block.Data())
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	rewardHeight := snapshotBlock.Height()
	locHash := getKey(block.AccountAddress(), *gid)
	oldData := vm.Db.Storage(block.ToAddress(), locHash)
	if len(oldData) > 0 {
		old := new(VariableRegistration)
		err := p.UnpackVariable(old, VariableNameRegistration, vm.Db.Storage(block.ToAddress(), locHash))
		if err != nil || old.Amount.Sign() > 0 {
			// duplicate register
			return ErrInvalidData
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
	}
	registerInfo, _ := p.PackVariable(VariableNameRegistration, block.Amount(), snapshotBlock.Timestamp(), rewardHeight, common.Big0)
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	return nil
}
func (p *register) doReceiveCancelRegister(vm *VM, block VmAccountBlock) error {
	gid := new(types.Gid)
	p.UnpackMethod(gid, MethodNameCancelRegister, block.Data())

	locHash := getKey(block.AccountAddress(), *gid)
	old := new(VariableRegistration)
	err := p.UnpackVariable(old, VariableNameRegistration, vm.Db.Storage(block.ToAddress(), locHash))
	if err != nil || old.Amount.Sign() == 0 {
		return ErrInvalidData
	}

	// update lock amount and loc start timestamp
	snapshotBlock := vm.Db.SnapshotBlock(block.SnapshotHash())
	registerInfo, _ := p.PackVariable(VariableNameRegistration, common.Big0, int64(0), old.RewardHeight, snapshotBlock.Height())
	vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	// return locked ViteToken
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(old.Amount)
	refundBlock.SetTokenId(util.ViteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), util.Big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}
func (p *register) doReceiveReward(vm *VM, block VmAccountBlock) error {
	param := new(ParamReward)
	p.UnpackMethod(param, MethodNameReward, block.Data())
	locHash := getKey(block.AccountAddress(), param.Gid)
	old := new(VariableRegistration)
	err := p.UnpackVariable(old, VariableNameRegistration, vm.Db.Storage(block.ToAddress(), locHash))
	if err != nil || old.RewardHeight.Cmp(param.StartHeight) != 0 {
		return ErrInvalidData
	}
	if old.CancelHeight.Sign() > 0 {
		switch param.EndHeight.Cmp(old.CancelHeight) {
		case 1:
			return ErrInvalidData
		case 0:
			// delete storage when register canceled and reward drained
			vm.Db.SetStorage(block.ToAddress(), locHash, nil)
		case -1:
			// get reward partly, update storage
			registerInfo, _ := p.PackVariable(VariableNameRegistration, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
			vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
		}
	} else {
		registerInfo, _ := p.PackVariable(VariableNameRegistration, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
		vm.Db.SetStorage(block.ToAddress(), locHash, registerInfo)
	}

	if param.Amount.Sign() > 0 {
		// create reward and return
		refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendReward, block.Depth()+1)
		refundBlock.SetAmount(param.Amount)
		refundBlock.SetTokenId(util.ViteTokenTypeId)
		refundBlock.SetHeight(new(big.Int).Add(block.Height(), util.Big1))
		vm.blockList = append(vm.blockList, refundBlock)
	}
	return nil
}

type vote struct {
	abi.ABIContract
}

func (p *vote) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return big.NewInt(0)
}

func (p *vote) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if method, err := p.MethodById(block.Data()[0:4]); err == nil {
		switch method.Name {
		case MethodNameVote:
			return p.doSendVote(vm, block, quotaLeft)
		case MethodNameCancelVote:
			return p.doSendCancelVote(vm, block, quotaLeft)
		}
	}
	return quotaLeft, ErrInvalidData
}

// vote for a super node of a consensus group
func (p *vote) doSendVote(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, voteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamVote)
	err = p.UnpackMethod(param, MethodNameVote, block.Data())
	if err != nil || !isExistGid(vm.Db, param.Gid) || !vm.Db.IsExistAddress(param.Node) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel vote for a super node of a consensus group
func (p *vote) doSendCancelVote(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	gid := new(types.Gid)
	err = p.UnpackMethod(gid, MethodNameCancelVote, block.Data())
	if err != nil || !isExistGid(vm.Db, *gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}
func (p *vote) doReceive(vm *VM, block VmAccountBlock) error {
	if method, err := p.MethodById(block.Data()[0:4]); err == nil {
		switch method.Name {
		case MethodNameVote:
			return p.doReceiveVote(vm, block)
		case MethodNameCancelVote:
			return p.doReceiveCancelVote(vm, block)
		}
	}
	return ErrInvalidData
}
func (p *vote) doReceiveVote(vm *VM, block VmAccountBlock) error {
	param := new(ParamVote)
	p.UnpackMethod(param, MethodNameVote, block.Data())
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := getKey(block.AccountAddress(), param.Gid)
	// storage value: superNodeAddress(0:32)
	voteStatus, _ := p.PackVariable(VariableNameVoteStatus, param.Node)
	vm.Db.SetStorage(block.ToAddress(), locHash, voteStatus)
	return nil
}
func (p *vote) doReceiveCancelVote(vm *VM, block VmAccountBlock) error {
	gid := new(types.Gid)
	p.UnpackMethod(gid, MethodNameCancelVote, block.Data())
	locHash := getKey(block.AccountAddress(), *gid)
	vm.Db.SetStorage(block.ToAddress(), locHash, nil)
	return nil
}

type pledge struct {
	abi.ABIContract
}

func (p *pledge) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return big.NewInt(0)
}

func (p *pledge) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	if method, err := p.MethodById(block.Data()[0:4]); err == nil {
		switch method.Name {
		case MethodNamePledge:
			return p.doSendPledge(vm, block, quotaLeft)
		case MethodNameCancelPledge:
			return p.doSendCancelPledge(vm, block, quotaLeft)
		}
	}
	return quotaLeft, ErrInvalidData
}

// pledge ViteToken for a beneficial to get quota
func (p *pledge) doSendPledge(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, pledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() == 0 ||
		!util.IsViteToken(block.TokenId()) ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamPledge)
	err = p.UnpackMethod(param, MethodNamePledge, block.Data())
	if err != nil || !vm.Db.IsExistAddress(param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}

	if param.WithdrawTime < vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp()+pledgeTime {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel pledge ViteToken
func (p *pledge) doSendCancelPledge(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, cancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() > 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamCancelPledge)
	err = p.UnpackMethod(param, MethodNameCancelPledge, block.Data())
	if err != nil || !vm.Db.IsExistAddress(param.Beneficial) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

func (p *pledge) doReceive(vm *VM, block VmAccountBlock) error {
	if method, err := p.MethodById(block.Data()[0:4]); err == nil {
		switch method.Name {
		case MethodNamePledge:
			return p.doReceivePledge(vm, block)
		case MethodNameCancelPledge:
			return p.doReceiveCancelPledge(vm, block)
		}
	}
	return ErrInvalidData
}

func (p *pledge) doReceivePledge(vm *VM, block VmAccountBlock) error {
	param := new(ParamPledge)
	p.UnpackMethod(param, MethodNamePledge, block.Data())
	// storage key for pledge beneficial: hash(beneficial)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes())
	// storage key for pledge: hash(owner, hash(beneficial))
	locHashPledge := types.DataHash(append(block.AccountAddress().Bytes(), locHashBeneficial.Bytes()...))
	// storage value for pledge: pledge amount(0:32) + withdrawTime(32:64)
	oldPledgeData := vm.Db.Storage(block.ToAddress(), locHashPledge)
	amount := new(big.Int)
	if len(oldPledgeData) > 0 {
		oldPledge := new(VariablePledgeInfo)
		p.UnpackVariable(oldPledge, VariableNamePledgeInfo, oldPledgeData)
		if param.WithdrawTime < oldPledge.WithdrawTime {
			return ErrInvalidData
		}
		amount = oldPledge.Amount
	}
	amount.Add(amount, block.Amount())
	pledgeInfo, _ := p.PackVariable(VariableNamePledgeInfo, amount, param.WithdrawTime)
	vm.Db.SetStorage(block.ToAddress(), locHashPledge, pledgeInfo)

	// storage value for quota: quota amount(0:32)
	oldBeneficialData := vm.Db.Storage(block.ToAddress(), locHashBeneficial)
	beneficialAmount := new(big.Int)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(VariablePledgeBeneficial)
		p.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, block.Amount())
	beneficialData, _ := p.PackVariable(VariableNamePledgeBeneficial, beneficialAmount)
	vm.Db.SetStorage(block.ToAddress(), locHashBeneficial, beneficialData)
	return nil
}
func (p *pledge) doReceiveCancelPledge(vm *VM, block VmAccountBlock) error {
	param := new(ParamCancelPledge)
	p.UnpackMethod(param, MethodNameCancelPledge, block.Data())
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes())
	locHashPledge := types.DataHash(append(block.AccountAddress().Bytes(), locHashBeneficial.Bytes()...))
	oldPledge := new(VariablePledgeInfo)
	err := p.UnpackVariable(oldPledge, VariableNamePledgeInfo, vm.Db.Storage(block.ToAddress(), locHashPledge))
	if err != nil || oldPledge.WithdrawTime > vm.Db.SnapshotBlock(block.SnapshotHash()).Timestamp() || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(VariablePledgeBeneficial)
	err = p.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, vm.Db.Storage(block.ToAddress(), locHashBeneficial))
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		vm.Db.SetStorage(block.ToAddress(), locHashPledge, nil)
	} else {
		pledgeInfo, _ := p.PackVariable(VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawTime)
		vm.Db.SetStorage(block.ToAddress(), locHashPledge, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		vm.Db.SetStorage(block.ToAddress(), locHashBeneficial, nil)
	} else {
		pledgeBeneficial, _ := p.PackVariable(VariableNamePledgeBeneficial, oldBeneficial.Amount)
		vm.Db.SetStorage(block.ToAddress(), locHashBeneficial, pledgeBeneficial)
	}

	// append refund block
	refundBlock := vm.createBlock(block.ToAddress(), block.AccountAddress(), BlockTypeSendCall, block.Depth()+1)
	refundBlock.SetAmount(param.Amount)
	refundBlock.SetTokenId(util.ViteTokenTypeId)
	refundBlock.SetHeight(new(big.Int).Add(block.Height(), util.Big1))
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}

type consensusGroup struct {
	abi.ABIContract
}

func (p *consensusGroup) createFee(vm *VM, block VmAccountBlock) *big.Int {
	return new(big.Int).Set(createConsensusGroupFee)
}

// create consensus group
func (p *consensusGroup) doSend(vm *VM, block VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, createConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount().Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress()) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamCreateConsensusGroup)
	err = p.UnpackMethod(param, MethodNameCreateConsensusGroup, block.Data())
	if err != nil {
		return quotaLeft, err
	}
	if err := checkCreateConsensusGroupData(vm, param); err != nil {
		return quotaLeft, err
	}
	// data: methodSelector(0:4) + gid(4:36) + ConsensusGroup
	gid := types.DataToGid(block.AccountAddress().Bytes(), block.Height().Bytes(), block.PrevHash().Bytes(), block.SnapshotHash().Bytes())
	if util.AllZero(gid.Bytes()) || isExistGid(vm.Db, gid) {
		return quotaLeft, ErrInvalidData
	}
	paramData, _ := p.PackMethod(MethodNameCreateConsensusGroup, gid, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	block.SetData(paramData)
	quotaLeft, err = useQuotaForData(block.Data(), quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func checkCreateConsensusGroupData(vm *VM, param *ParamCreateConsensusGroup) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax {
		return ErrInvalidData
	}
	if err := checkCondition(vm, param.CountingRuleId, param.CountingRuleParam, "counting"); err != nil {
		return ErrInvalidData
	}
	if err := checkCondition(vm, param.RegisterConditionId, param.RegisterConditionParam, "register"); err != nil {
		return ErrInvalidData
	}
	if err := checkCondition(vm, param.VoteConditionId, param.VoteConditionParam, "vote"); err != nil {
		return ErrInvalidData
	}
	return nil
}
func checkCondition(vm *VM, conditionId uint8, conditionParam []byte, conditionIdPrefix string) error {
	condition, ok := SimpleCountingRuleList[CountingRuleCode(conditionIdPrefix+strconv.Itoa(int(conditionId)))]
	if !ok {
		return ErrInvalidData
	}
	if len(conditionParam) > 0 {
		if ok := condition.checkParam(conditionParam, vm.Db); !ok {
			return ErrInvalidData
		}
		return nil
	}
	return nil
}

func (p *consensusGroup) doReceive(vm *VM, block VmAccountBlock) error {
	param := new(ParamCreateConsensusGroup)
	p.UnpackMethod(param, MethodNameCreateConsensusGroup, block.Data())
	locHash := types.DataHash(param.Gid.Bytes())
	if len(vm.Db.Storage(block.ToAddress(), locHash)) > 0 {
		return ErrIdCollision
	}
	groupInfo, _ := p.PackVariable(VariableNameConsensusGroupInfo, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	vm.Db.SetStorage(block.ToAddress(), locHash, groupInfo)
	return nil
}
func isUserAccount(db VmDatabase, addr types.Address) bool {
	return len(db.ContractCode(addr)) == 0
}

func getKey(addr types.Address, gid types.Gid) types.Hash {
	var data = types.Hash{}
	copy(data[2:12], gid[:])
	copy(data[12:], addr[:])
	return data
}
