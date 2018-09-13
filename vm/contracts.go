package vm

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strconv"
	"time"
)

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressPledge, _         = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
)

type precompiledContract interface {
	createFee(vm *VM, block *ledger.AccountBlock) *big.Int
	doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error)
	doReceive(vm *VM, block *ledger.AccountBlock) error
}

var simpleContracts = map[types.Address]precompiledContract{
	AddressRegister:       &register{},
	AddressVote:           &vote{},
	AddressPledge:         &pledge{},
	AddressConsensusGroup: &consensusGroup{},
}

func getPrecompiledContract(address types.Address) (precompiledContract, bool) {
	p, ok := simpleContracts[address]
	return p, ok
}

type register struct {
}

func (p *register) createFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}
func (p *register) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if method, err := ABI_register.MethodById(block.Data[0:4]); err == nil {
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

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *register) doSendRegister(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(registerAmount) != 0 ||
		!util.IsViteToken(block.TokenId) ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}

	gid := new(types.Gid)
	err = ABI_register.UnpackMethod(gid, MethodNameRegister, block.Data)
	if err != nil || !isExistGid(vm.Db, *gid) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *register) doSendCancelRegister(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
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
	gid := new(types.Gid)
	err = ABI_register.UnpackMethod(gid, MethodNameCancelRegister, block.Data)
	if err != nil || !isExistGid(vm.Db, *gid) {
		return quotaLeft, ErrInvalidData
	}

	locHash := getKey(block.AccountAddress, *gid)
	old := new(VariableRegistration)
	err = ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, locHash))
	if err != nil || time.Unix(old.Timestamp+registerLockTime, 0).Before(*vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// get reward of generating snapshot block
func (p *register) doSendReward(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
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
	if err != nil || !util.IsSnapshotGid(param.Gid) {
		return quotaLeft, ErrInvalidData
	}
	locHash := getKey(block.AccountAddress, param.Gid)
	old := new(VariableRegistration)
	err = ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, locHash))
	if err != nil {
		return quotaLeft, ErrInvalidData
	}
	// newRewardHeight := min(currentSnapshotHeight-50, userDefined, cancelSnapshotHeight)
	newRewardHeight := vm.Db.GetSnapshotBlock(&block.SnapshotHash).Height - rewardHeightLimit
	if param.EndHeight > 0 {
		newRewardHeight = util.Min(newRewardHeight, param.EndHeight)
	}
	if old.CancelHeight > 0 {
		newRewardHeight = util.Min(newRewardHeight, old.CancelHeight)
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
	data, err := ABI_register.PackMethod(MethodNameReward, param.Gid, newRewardHeight, old.RewardHeight, param.Amount)
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

func (p *register) doReceive(vm *VM, block *ledger.AccountBlock) error {
	if method, err := ABI_register.MethodById(block.Data[0:4]); err == nil {
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
func (p *register) doReceiveRegister(vm *VM, block *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABI_register.UnpackMethod(gid, MethodNameRegister, block.Data)
	snapshotBlock := vm.Db.GetSnapshotBlock(&block.SnapshotHash)
	rewardHeight := snapshotBlock.Height
	locHash := getKey(block.AccountAddress, *gid)
	oldData := vm.Db.GetStorage(&block.ToAddress, locHash)
	if len(oldData) > 0 {
		old := new(VariableRegistration)
		err := ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, locHash))
		if err != nil || old.Timestamp > 0 {
			// duplicate register
			return ErrInvalidData
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
	}
	registerInfo, err := ABI_register.PackVariable(VariableNameRegistration, block.Amount, snapshotBlock.Timestamp.Unix(), rewardHeight, uint64(0))
	if err != nil {
		fmt.Println(err)
	}
	vm.Db.SetStorage(locHash, registerInfo)
	return nil
}
func (p *register) doReceiveCancelRegister(vm *VM, block *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABI_register.UnpackMethod(gid, MethodNameCancelRegister, block.Data)

	locHash := getKey(block.AccountAddress, *gid)
	old := new(VariableRegistration)
	err := ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, locHash))
	if err != nil || old.Timestamp == 0 {
		return ErrInvalidData
	}

	// update lock amount and loc start timestamp
	snapshotBlock := vm.Db.GetSnapshotBlock(&block.SnapshotHash)
	registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, common.Big0, int64(0), old.RewardHeight, snapshotBlock.Height)
	vm.Db.SetStorage(locHash, registerInfo)
	// return locked ViteToken
	if old.Amount.Sign() > 0 {
		refundBlock := &ledger.AccountBlock{AccountAddress: block.ToAddress, ToAddress: block.AccountAddress, BlockType: ledger.BlockTypeSendCall, Amount: old.Amount, TokenId: *ledger.ViteTokenId(), Height: block.Height + 1}
		vm.blockList = append(vm.blockList, refundBlock)
	}
	return nil
}
func (p *register) doReceiveReward(vm *VM, block *ledger.AccountBlock) error {
	param := new(ParamReward)
	ABI_register.UnpackMethod(param, MethodNameReward, block.Data)
	locHash := getKey(block.AccountAddress, param.Gid)
	old := new(VariableRegistration)
	err := ABI_register.UnpackVariable(old, VariableNameRegistration, vm.Db.GetStorage(&block.ToAddress, locHash))
	if err != nil || old.RewardHeight != param.StartHeight {
		return ErrInvalidData
	}
	if old.CancelHeight > 0 {
		if param.EndHeight > old.CancelHeight {
			return ErrInvalidData
		} else if param.EndHeight == old.CancelHeight {
			// delete storage when register canceled and reward drained
			vm.Db.SetStorage(locHash, nil)
		} else {
			// get reward partly, update storage
			registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
			vm.Db.SetStorage(locHash, registerInfo)
		}
	} else {
		registerInfo, _ := ABI_register.PackVariable(VariableNameRegistration, old.Amount, old.Timestamp, param.EndHeight, old.CancelHeight)
		vm.Db.SetStorage(locHash, registerInfo)
	}

	if param.Amount.Sign() > 0 {
		// create reward and return
		refundBlock := &ledger.AccountBlock{AccountAddress: block.ToAddress, ToAddress: block.AccountAddress, BlockType: ledger.BlockTypeSendReward, Amount: param.Amount, TokenId: *ledger.ViteTokenId(), Height: block.Height + 1}
		vm.blockList = append(vm.blockList, refundBlock)
	}
	return nil
}

type vote struct {
}

func (p *vote) createFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

func (p *vote) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if method, err := ABI_vote.MethodById(block.Data[0:4]); err == nil {
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
func (p *vote) doSendVote(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
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
	if err != nil || !isExistGid(vm.Db, param.Gid) || !vm.Db.IsAddressExisted(&param.Node) {
		return quotaLeft, ErrInvalidData
	}
	return quotaLeft, nil
}

// cancel vote for a super node of a consensus group
func (p *vote) doSendCancelVote(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
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
func (p *vote) doReceive(vm *VM, block *ledger.AccountBlock) error {
	if method, err := ABI_vote.MethodById(block.Data[0:4]); err == nil {
		switch method.Name {
		case MethodNameVote:
			return p.doReceiveVote(vm, block)
		case MethodNameCancelVote:
			return p.doReceiveCancelVote(vm, block)
		}
	}
	return ErrInvalidData
}
func (p *vote) doReceiveVote(vm *VM, block *ledger.AccountBlock) error {
	param := new(ParamVote)
	ABI_vote.UnpackMethod(param, MethodNameVote, block.Data)
	// storage key: 00(0:2) + gid(2:12) + voter address(12:32)
	locHash := getKey(block.AccountAddress, param.Gid)
	voteStatus, _ := ABI_vote.PackVariable(VariableNameVoteStatus, param.Node)
	vm.Db.SetStorage(locHash, voteStatus)
	return nil
}
func (p *vote) doReceiveCancelVote(vm *VM, block *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABI_vote.UnpackMethod(gid, MethodNameCancelVote, block.Data)
	locHash := getKey(block.AccountAddress, *gid)
	vm.Db.SetStorage(locHash, nil)
	return nil
}

type pledge struct{}

func (p *pledge) createFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return big.NewInt(0)
}

func (p *pledge) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	if method, err := ABI_pledge.MethodById(block.Data[0:4]); err == nil {
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
func (p *pledge) doSendPledge(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, pledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = useQuotaForData(block.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() == 0 ||
		!util.IsViteToken(block.TokenId) ||
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

// cancel pledge ViteToken
func (p *pledge) doSendCancelPledge(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
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

func (p *pledge) doReceive(vm *VM, block *ledger.AccountBlock) error {
	if method, err := ABI_pledge.MethodById(block.Data[0:4]); err == nil {
		switch method.Name {
		case MethodNamePledge:
			return p.doReceivePledge(vm, block)
		case MethodNameCancelPledge:
			return p.doReceiveCancelPledge(vm, block)
		}
	}
	return ErrInvalidData
}

func (p *pledge) doReceivePledge(vm *VM, block *ledger.AccountBlock) error {
	param := new(ParamPledge)
	ABI_pledge.UnpackMethod(param, MethodNamePledge, block.Data)
	// storage key for pledge beneficial: hash(beneficial)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes()).Bytes()
	// storage key for pledge: hash(owner, hash(beneficial))
	locHashPledge := types.DataHash(append(block.AccountAddress.Bytes(), locHashBeneficial...)).Bytes()
	oldPledgeData := vm.Db.GetStorage(&block.ToAddress, locHashPledge)
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
	oldBeneficialData := vm.Db.GetStorage(&block.ToAddress, locHashBeneficial)
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
func (p *pledge) doReceiveCancelPledge(vm *VM, block *ledger.AccountBlock) error {
	param := new(ParamCancelPledge)
	ABI_pledge.UnpackMethod(param, MethodNameCancelPledge, block.Data)
	locHashBeneficial := types.DataHash(param.Beneficial.Bytes()).Bytes()
	locHashPledge := types.DataHash(append(block.AccountAddress.Bytes(), locHashBeneficial...)).Bytes()
	oldPledge := new(VariablePledgeInfo)
	err := ABI_pledge.UnpackVariable(oldPledge, VariableNamePledgeInfo, vm.Db.GetStorage(&block.ToAddress, locHashPledge))
	if err != nil || time.Unix(oldPledge.WithdrawTime, 0).After(*vm.Db.GetSnapshotBlock(&block.SnapshotHash).Timestamp) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return ErrInvalidData
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(VariablePledgeBeneficial)
	err = ABI_pledge.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, vm.Db.GetStorage(&block.ToAddress, locHashBeneficial))
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
	refundBlock := &ledger.AccountBlock{AccountAddress: block.ToAddress, ToAddress: block.AccountAddress, BlockType: ledger.BlockTypeSendCall, Amount: param.Amount, TokenId: *ledger.ViteTokenId(), Height: block.Height + 1}
	vm.blockList = append(vm.blockList, refundBlock)
	return nil
}

type consensusGroup struct{}

func (p *consensusGroup) createFee(vm *VM, block *ledger.AccountBlock) *big.Int {
	return new(big.Int).Set(createConsensusGroupFee)
}

// create consensus group
func (p *consensusGroup) doSend(vm *VM, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := useQuota(quotaLeft, createConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!isUserAccount(vm.Db, block.AccountAddress) {
		return quotaLeft, ErrInvalidData
	}
	param := new(ParamCreateConsensusGroup)
	err = ABI_consensusGroup.UnpackMethod(param, MethodNameCreateConsensusGroup, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := p.checkCreateConsensusGroupData(vm, param); err != nil {
		return quotaLeft, err
	}
	// data: methodSelector(0:4) + gid(4:36) + ConsensusGroup
	gid := types.DataToGid(block.AccountAddress.Bytes(), new(big.Int).SetUint64(block.Height).Bytes(), block.PrevHash.Bytes(), block.SnapshotHash.Bytes())
	if util.AllZero(gid.Bytes()) || isExistGid(vm.Db, gid) {
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
func (p *consensusGroup) checkCreateConsensusGroupData(vm *VM, param *ParamCreateConsensusGroup) error {
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
func (p *consensusGroup) checkCondition(vm *VM, conditionId uint8, conditionParam []byte, conditionIdPrefix string) error {
	condition, ok := SimpleCountingRuleList[CountingRuleCode(conditionIdPrefix+strconv.Itoa(int(conditionId)))]
	if !ok {
		return ErrInvalidData
	}
	if ok := condition.checkParam(conditionParam, vm.Db); !ok {
		return ErrInvalidData
	}
	return nil
}
func (p *consensusGroup) doReceive(vm *VM, block *ledger.AccountBlock) error {
	param := new(ParamCreateConsensusGroup)
	ABI_consensusGroup.UnpackMethod(param, MethodNameCreateConsensusGroup, block.Data)
	locHash := types.DataHash(param.Gid.Bytes()).Bytes()
	if len(vm.Db.GetStorage(&block.ToAddress, locHash)) > 0 {
		return ErrIdCollision
	}
	groupInfo, _ := ABI_consensusGroup.PackVariable(VariableNameConsensusGroupInfo, param.NodeCount, param.Interval, param.CountingRuleId, param.CountingRuleParam, param.RegisterConditionId, param.RegisterConditionParam, param.VoteConditionId, param.VoteConditionParam)
	vm.Db.SetStorage(locHash, groupInfo)
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
	if err != nil || db.GetToken(v) == nil {
		return false
	}
	return true
}

type registerConditionOfSnapshot struct{}

func (c registerConditionOfSnapshot) checkParam(param []byte, db VmDatabase) bool {
	v := new(VariableConditionRegister1)
	err := ABI_consensusGroup.UnpackVariable(v, VariableNameConditionRegister1, param)
	if err != nil || db.GetToken(&v.PledgeToken) == nil {
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
	if err != nil || db.GetToken(&v.KeepToken) == nil {
		return false
	}
	return true
}

func isUserAccount(db VmDatabase, addr types.Address) bool {
	return len(db.GetContractCode(&addr)) == 0
}

func getKey(addr types.Address, gid types.Gid) []byte {
	var data = make([]byte, types.HashSize)
	copy(data[2:12], gid[:])
	copy(data[12:], addr[:])
	return data
}

func getAddr(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[12:])
	return addr
}
