package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"regexp"
)

type MethodCreateConsensusGroup struct{}

func (p *MethodCreateConsensusGroup) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCreateConsensusGroup) GetRefundData() []byte {
	return []byte{1}
}

func (p *MethodCreateConsensusGroup) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CreateConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!util.IsViteToken(block.TokenId) ||
		!util.IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(types.ConsensusGroupInfo)
	err = cabi.ABIConsensusGroup.UnpackMethod(param, cabi.MethodNameCreateConsensusGroup, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := CheckCreateConsensusGroupData(db, param); err != nil {
		return quotaLeft, err
	}
	gid := cabi.NewGid(block.AccountAddress, block.Height, block.PrevHash, block.SnapshotHash)
	if IsExistGid(db, gid) {
		return quotaLeft, errors.New("consensus group id already exists")
	}
	paramData, _ := cabi.ABIConsensusGroup.PackMethod(
		cabi.MethodNameCreateConsensusGroup,
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
	block.Data = paramData
	return quotaLeft, nil
}
func CheckCreateConsensusGroupData(db vmctxt_interface.VmDatabase, param *types.ConsensusGroupInfo) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax ||
		param.PerCount < cgPerCountMin || param.PerCount > cgPerCountMax ||
		// no overflow
		param.PerCount*param.Interval < cgPerIntervalMin || param.PerCount*param.Interval > cgPerIntervalMax ||
		param.RandCount > param.NodeCount ||
		(param.RandCount > 0 && param.RandRank < param.NodeCount) {
		return errors.New("invalid consensus group param")
	}
	if cabi.GetTokenById(db, param.CountingTokenId) == nil {
		return errors.New("counting token id not exist")
	}
	if err := checkCondition(db, param.RegisterConditionId, param.RegisterConditionParam, cabi.RegisterConditionPrefix); err != nil {
		return err
	}
	if err := checkCondition(db, param.VoteConditionId, param.VoteConditionParam, cabi.VoteConditionPrefix); err != nil {
		return err
	}
	return nil
}
func checkCondition(db vmctxt_interface.VmDatabase, conditionId uint8, conditionParam []byte, conditionIdPrefix cabi.ConditionCode) error {
	condition, ok := getConsensusGroupCondition(conditionId, conditionIdPrefix)
	if !ok {
		return errors.New("condition id not exist")
	}
	if ok := condition.checkParam(conditionParam, db); !ok {
		return errors.New("invalid condition param")
	}
	return nil
}
func (p *MethodCreateConsensusGroup) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(types.ConsensusGroupInfo)
	cabi.ABIConsensusGroup.UnpackMethod(param, cabi.MethodNameCreateConsensusGroup, sendBlock.Data)
	key := cabi.GetConsensusGroupKey(param.Gid)
	if len(db.GetStorage(&block.AccountAddress, key)) > 0 {
		return nil, util.ErrIdCollision
	}
	groupInfo, _ := cabi.ABIConsensusGroup.PackVariable(
		cabi.VariableNameConsensusGroupInfo,
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
		db.CurrentSnapshotBlock().Height+nodeConfig.params.CreateConsensusGroupPledgeHeight)
	db.SetStorage(key, groupInfo)
	return nil, nil
}

type MethodCancelConsensusGroup struct{}

func (p *MethodCancelConsensusGroup) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelConsensusGroup) GetRefundData() []byte {
	return []byte{2}
}

// Cancel consensus group and get pledge back.
// A canceled consensus group(no-active) will not generate contract blocks after cancel receive block is confirmed.
// Consensus group name is kept even if canceled.
func (p *MethodCancelConsensusGroup) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	err = cabi.ABIConsensusGroup.UnpackMethod(gid, cabi.MethodNameCancelConsensusGroup, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	groupInfo := cabi.GetConsensusGroup(db, *gid)
	if groupInfo == nil ||
		block.AccountAddress != groupInfo.Owner ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > db.CurrentSnapshotBlock().Height {
		return quotaLeft, errors.New("invalid group or owner or not due yet")
	}
	return quotaLeft, nil
}
func (p *MethodCancelConsensusGroup) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	gid := new(types.Gid)
	cabi.ABIConsensusGroup.UnpackMethod(gid, cabi.MethodNameCancelConsensusGroup, sendBlock.Data)
	key := cabi.GetConsensusGroupKey(*gid)
	groupInfo := cabi.GetConsensusGroup(db, *gid)
	if groupInfo == nil ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > db.CurrentSnapshotBlock().Height {
		return nil, errors.New("pledge not yet due")
	}
	newGroupInfo, _ := cabi.ABIConsensusGroup.PackVariable(
		cabi.VariableNameConsensusGroupInfo,
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
	db.SetStorage(key, newGroupInfo)
	if groupInfo.PledgeAmount.Sign() > 0 {
		return []*SendBlock{
			{
				block,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				groupInfo.PledgeAmount,
				ledger.ViteTokenId,
				[]byte{},
			},
		}, nil
	}
	return nil, nil
}

type MethodReCreateConsensusGroup struct{}

func (p *MethodReCreateConsensusGroup) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodReCreateConsensusGroup) GetRefundData() []byte {
	return []byte{3}
}

// Pledge again for a canceled consensus group.
// A consensus group will start generate contract blocks after recreate receive block is confirmed.
func (p *MethodReCreateConsensusGroup) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, ReCreateConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!util.IsViteToken(block.TokenId) ||
		!util.IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	if err = cabi.ABIConsensusGroup.UnpackMethod(gid, cabi.MethodNameReCreateConsensusGroup, block.Data); err != nil {
		return quotaLeft, err
	}
	if groupInfo := cabi.GetConsensusGroup(db, *gid); groupInfo == nil ||
		block.AccountAddress != groupInfo.Owner ||
		groupInfo.IsActive() {
		return quotaLeft, errors.New("invalid group info or owner or status")
	}
	return quotaLeft, nil
}
func (p *MethodReCreateConsensusGroup) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	gid := new(types.Gid)
	cabi.ABIConsensusGroup.UnpackMethod(gid, cabi.MethodNameReCreateConsensusGroup, sendBlock.Data)
	key := cabi.GetConsensusGroupKey(*gid)
	groupInfo := cabi.GetConsensusGroup(db, *gid)
	if groupInfo == nil ||
		groupInfo.IsActive() {
		return nil, errors.New("consensus group is active")
	}
	newGroupInfo, _ := cabi.ABIConsensusGroup.PackVariable(
		cabi.VariableNameConsensusGroupInfo,
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
		db.CurrentSnapshotBlock().Height+nodeConfig.params.CreateConsensusGroupPledgeHeight)
	db.SetStorage(key, newGroupInfo)
	return nil, nil
}

type createConsensusGroupCondition interface {
	checkParam(param []byte, db vmctxt_interface.VmDatabase) bool
	checkData(paramData []byte, db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, blockParamInterface interface{}, method string) bool
}

var SimpleCountingRuleList = map[cabi.ConditionCode]createConsensusGroupCondition{
	cabi.RegisterConditionOfPledge: &registerConditionOfPledge{},
	cabi.VoteConditionOfDefault:    &voteConditionOfDefault{},
	cabi.VoteConditionOfBalance:    &voteConditionOfKeepToken{},
}

func getConsensusGroupCondition(conditionId uint8, conditionIdPrefix cabi.ConditionCode) (createConsensusGroupCondition, bool) {
	condition, ok := SimpleCountingRuleList[conditionIdPrefix+cabi.ConditionCode(conditionId)]
	return condition, ok
}

func getRegisterWithdrawHeightByCondition(conditionId uint8, param []byte, currentHeight uint64) uint64 {
	if cabi.RegisterConditionPrefix+cabi.ConditionCode(conditionId) == cabi.RegisterConditionOfPledge {
		v := new(cabi.VariableConditionRegisterOfPledge)
		cabi.ABIConsensusGroup.UnpackVariable(v, cabi.VariableNameConditionRegisterOfPledge, param)
		return currentHeight + v.PledgeHeight
	}
	return currentHeight
}

type registerConditionOfPledge struct{}

func (c registerConditionOfPledge) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(cabi.VariableConditionRegisterOfPledge)
	err := cabi.ABIConsensusGroup.UnpackVariable(v, cabi.VariableNameConditionRegisterOfPledge, param)
	if err != nil ||
		cabi.GetTokenById(db, v.PledgeToken) == nil ||
		v.PledgeAmount.Sign() == 0 ||
		v.PledgeHeight < nodeConfig.params.MinPledgeHeight {
		return false
	}
	return true
}

func (c registerConditionOfPledge) checkData(paramData []byte, db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, blockParamInterface interface{}, method string) bool {
	switch method {
	case cabi.MethodNameRegister:
		blockParam := blockParamInterface.(*cabi.ParamRegister)
		if blockParam.Gid == types.DELEGATE_GID ||
			len(blockParam.Name) == 0 ||
			len(blockParam.Name) > registrationNameLengthMax {
			return false
		}
		if ok, _ := regexp.MatchString("^([0-9a-zA-Z_.]+[ ]?)*[0-9a-zA-Z_.]$", blockParam.Name); !ok {
			return false
		}
		param := new(cabi.VariableConditionRegisterOfPledge)
		cabi.ABIConsensusGroup.UnpackVariable(param, cabi.VariableNameConditionRegisterOfPledge, paramData)
		if block.Amount.Cmp(param.PledgeAmount) != 0 || block.TokenId != param.PledgeToken {
			return false
		}
	case cabi.MethodNameCancelRegister:
		if block.Amount.Sign() != 0 ||
			!util.IsUserAccount(db, block.AccountAddress) {
			return false
		}

		param := new(cabi.VariableConditionRegisterOfPledge)
		cabi.ABIConsensusGroup.UnpackVariable(param, cabi.VariableNameConditionRegisterOfPledge, paramData)
	case cabi.MethodNameUpdateRegistration:
		if block.Amount.Sign() != 0 {
			return false
		}
		blockParam := blockParamInterface.(*cabi.ParamRegister)
		if blockParam.Gid == types.DELEGATE_GID {
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
func (c voteConditionOfDefault) checkData(paramData []byte, db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, blockParamInterface interface{}, method string) bool {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db, block.AccountAddress) {
		return false
	}
	return true
}

type voteConditionOfKeepToken struct{}

func (c voteConditionOfKeepToken) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(cabi.VariableConditionVoteOfKeepToken)
	err := cabi.ABIConsensusGroup.UnpackVariable(v, cabi.VariableNameConditionVoteOfKeepToken, param)
	if err != nil || cabi.GetTokenById(db, v.KeepToken) == nil || v.KeepAmount.Sign() == 0 {
		return false
	}
	return true
}
func (c voteConditionOfKeepToken) checkData(paramData []byte, db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, blockParamInterface interface{}, method string) bool {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db, block.AccountAddress) {
		return false
	}
	param := new(cabi.VariableConditionVoteOfKeepToken)
	cabi.ABIConsensusGroup.UnpackVariable(param, cabi.VariableNameConditionVoteOfKeepToken, paramData)
	if db.GetBalance(&block.AccountAddress, &param.KeepToken).Cmp(param.KeepAmount) < 0 {
		return false
	}
	return true
}
