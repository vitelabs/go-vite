package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"regexp"
	"strings"
	"time"
)

const (
	jsonConsensusGroup = `
	[
		{"type":"function","name":"CreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
		{"type":"function","name":"CancelConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"function","name":"ReCreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"registerOfPledge","inputs":[{"name":"pledgeAmount","type":"uint256"},{"name":"pledgeToken","type":"tokenId"},{"name":"pledgeHeight","type":"uint64"}]},
		{"type":"variable","name":"voteOfKeepToken","inputs":[{"name":"keepAmount","type":"uint256"},{"name":"keepToken","type":"tokenId"}]}
	]`

	MethodNameCreateConsensusGroup        = "CreateConsensusGroup"
	MethodNameCancelConsensusGroup        = "CancelConsensusGroup"
	MethodNameReCreateConsensusGroup      = "ReCreateConsensusGroup"
	VariableNameConsensusGroupInfo        = "consensusGroupInfo"
	VariableNameConditionRegisterOfPledge = "registerOfPledge"
	VariableNameConditionVoteOfKeepToken  = "voteOfKeepToken"
)

var (
	ABIConsensusGroup, _ = abi.JSONToABIContract(strings.NewReader(jsonConsensusGroup))
)

type VariableConditionRegisterOfPledge struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeHeight uint64
}
type VariableConditionVoteOfKeepToken struct {
	KeepAmount *big.Int
	KeepToken  types.TokenTypeId
}
type ConsensusGroupInfo struct {
	Gid                    types.Gid         // Consensus group id
	NodeCount              uint8             // Active miner count
	Interval               int64             // Timestamp gap between two continuous block
	PerCount               int64             // Continuous block generation interval count
	RandCount              uint8             // Random miner count
	RandRank               uint8             // Chose random miner with a rank limit of vote
	CountingTokenId        types.TokenTypeId // Token id for selecting miner through vote
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
	Owner                  types.Address
	PledgeAmount           *big.Int
	WithdrawHeight         uint64
}

func (groupInfo *ConsensusGroupInfo) IsActive() bool {
	return groupInfo.WithdrawHeight > 0
}

func GetConsensusGroupKey(gid types.Gid) []byte {
	return helper.LeftPadBytes(gid.Bytes(), types.HashSize)
}
func GetGidFromConsensusGroupKey(key []byte) types.Gid {
	gid, _ := types.BytesToGid(key[types.HashSize-types.GidSize:])
	return gid
}

func NewGid(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.Gid {
	return types.DataToGid(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func GetActiveConsensusGroupList(db StorageDatabase) []*ConsensusGroupInfo {
	defer monitor.LogTime("vm", "GetActiveConsensusGroupList", time.Now())
	iterator := db.NewStorageIterator(&AddressConsensusGroup, nil)
	consensusGroupInfoList := make([]*ConsensusGroupInfo, 0)
	if iterator == nil {
		return consensusGroupInfoList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		consensusGroupInfo := new(ConsensusGroupInfo)
		if err := ABIConsensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value); err == nil {
			if consensusGroupInfo.IsActive() {
				consensusGroupInfo.Gid = GetGidFromConsensusGroupKey(key)
				consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
			}
		}
	}
	return consensusGroupInfoList
}

func GetConsensusGroup(db StorageDatabase, gid types.Gid) *ConsensusGroupInfo {
	data := db.GetStorage(&AddressConsensusGroup, GetConsensusGroupKey(gid))
	if len(data) > 0 {
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABIConsensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo
	}
	return nil
}

type MethodCreateConsensusGroup struct{}

func (p *MethodCreateConsensusGroup) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCreateConsensusGroup) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CreateConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!util.IsViteToken(block.AccountBlock.TokenId) ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(ConsensusGroupInfo)
	err = ABIConsensusGroup.UnpackMethod(param, MethodNameCreateConsensusGroup, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := CheckCreateConsensusGroupData(block.VmContext, param); err != nil {
		return quotaLeft, err
	}
	gid := NewGid(block.AccountBlock.AccountAddress, block.AccountBlock.Height, block.AccountBlock.PrevHash, block.AccountBlock.SnapshotHash)
	if IsExistGid(block.VmContext, gid) {
		return quotaLeft, errors.New("consensus group id already exists")
	}
	paramData, _ := ABIConsensusGroup.PackMethod(
		MethodNameCreateConsensusGroup,
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
	quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func CheckCreateConsensusGroupData(db vmctxt_interface.VmDatabase, param *ConsensusGroupInfo) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax ||
		param.PerCount < cgPerCountMin || param.PerCount > cgPerCountMax ||
		// no overflow
		param.PerCount*param.Interval < cgPerIntervalMin || param.PerCount*param.Interval > cgPerIntervalMax ||
		param.RandCount > param.NodeCount ||
		(param.RandCount > 0 && param.RandRank < param.NodeCount) {
		return errors.New("invalid consensus group param")
	}
	if GetTokenById(db, param.CountingTokenId) == nil {
		return errors.New("counting token id not exist")
	}
	if err := checkCondition(db, param.RegisterConditionId, param.RegisterConditionParam, RegisterConditionPrefix); err != nil {
		return err
	}
	if err := checkCondition(db, param.VoteConditionId, param.VoteConditionParam, VoteConditionPrefix); err != nil {
		return err
	}
	return nil
}
func checkCondition(db vmctxt_interface.VmDatabase, conditionId uint8, conditionParam []byte, conditionIdPrefix ConditionCode) error {
	condition, ok := getConsensusGroupCondition(conditionId, conditionIdPrefix)
	if !ok {
		return errors.New("condition id not exist")
	}
	if ok := condition.checkParam(conditionParam, db); !ok {
		return errors.New("invalid condition param")
	}
	return nil
}
func (p *MethodCreateConsensusGroup) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ConsensusGroupInfo)
	ABIConsensusGroup.UnpackMethod(param, MethodNameCreateConsensusGroup, sendBlock.Data)
	key := GetConsensusGroupKey(param.Gid)
	if len(block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)) > 0 {
		return util.ErrIdCollision
	}
	groupInfo, _ := ABIConsensusGroup.PackVariable(
		VariableNameConsensusGroupInfo,
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

type MethodCancelConsensusGroup struct{}

func (p *MethodCancelConsensusGroup) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// Cancel consensus group and get pledge back.
// A canceled consensus group(no-active) will not generate contract blocks after cancel receive block is confirmed.
// Consensus group name is kept even if canceled.
func (p *MethodCancelConsensusGroup) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	err = ABIConsensusGroup.UnpackMethod(gid, MethodNameCancelConsensusGroup, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, err
	}
	groupInfo := GetConsensusGroup(block.VmContext, *gid)
	if groupInfo == nil ||
		block.AccountBlock.AccountAddress != groupInfo.Owner ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height {
		return quotaLeft, errors.New("invalid group or owner or not due yet")
	}
	return quotaLeft, nil
}
func (p *MethodCancelConsensusGroup) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABIConsensusGroup.UnpackMethod(gid, MethodNameCancelConsensusGroup, sendBlock.Data)
	key := GetConsensusGroupKey(*gid)
	groupInfo := GetConsensusGroup(block.VmContext, *gid)
	if groupInfo == nil ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height {
		return errors.New("pledge not yet due")
	}
	newGroupInfo, _ := ABIConsensusGroup.PackVariable(
		VariableNameConsensusGroupInfo,
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
		context.AppendBlock(
			&vm_context.VmAccountBlock{
				util.MakeSendBlock(
					block.AccountBlock,
					sendBlock.AccountAddress,
					ledger.BlockTypeSendCall,
					groupInfo.PledgeAmount,
					ledger.ViteTokenId,
					context.GetNewBlockHeight(block),
					[]byte{},
				),
				nil,
			})
	}
	return nil
}

type MethodReCreateConsensusGroup struct{}

func (p *MethodReCreateConsensusGroup) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// Pledge again for a canceled consensus group.
// A consensus group will start generate contract blocks after recreate receive block is confirmed.
func (p *MethodReCreateConsensusGroup) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, ReCreateConsensusGroupGas)
	if err != nil {
		return quotaLeft, err
	}
	if quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft); err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!util.IsViteToken(block.AccountBlock.TokenId) ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	if err = ABIConsensusGroup.UnpackMethod(gid, MethodNameReCreateConsensusGroup, block.AccountBlock.Data); err != nil {
		return quotaLeft, err
	}
	if groupInfo := GetConsensusGroup(block.VmContext, *gid); groupInfo == nil ||
		block.AccountBlock.AccountAddress != groupInfo.Owner ||
		groupInfo.IsActive() {
		return quotaLeft, errors.New("invalid group info or owner or status")
	}
	return quotaLeft, nil
}
func (p *MethodReCreateConsensusGroup) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABIConsensusGroup.UnpackMethod(gid, MethodNameReCreateConsensusGroup, sendBlock.Data)
	key := GetConsensusGroupKey(*gid)
	groupInfo := GetConsensusGroup(block.VmContext, *gid)
	if groupInfo == nil ||
		groupInfo.IsActive() {
		return errors.New("consensus group is active")
	}
	newGroupInfo, _ := ABIConsensusGroup.PackVariable(
		VariableNameConsensusGroupInfo,
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

var SimpleCountingRuleList = map[ConditionCode]createConsensusGroupCondition{
	RegisterConditionOfPledge: &registerConditionOfPledge{},
	VoteConditionOfDefault:    &voteConditionOfDefault{},
	VoteConditionOfBalance:    &voteConditionOfKeepToken{},
}

func getConsensusGroupCondition(conditionId uint8, conditionIdPrefix ConditionCode) (createConsensusGroupCondition, bool) {
	condition, ok := SimpleCountingRuleList[conditionIdPrefix+ConditionCode(conditionId)]
	return condition, ok
}

type registerConditionOfPledge struct{}

func (c registerConditionOfPledge) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(VariableConditionRegisterOfPledge)
	err := ABIConsensusGroup.UnpackVariable(v, VariableNameConditionRegisterOfPledge, param)
	if err != nil ||
		GetTokenById(db, v.PledgeToken) == nil ||
		v.PledgeAmount.Sign() == 0 ||
		v.PledgeHeight < minPledgeHeight {
		return false
	}
	return true
}

func (c registerConditionOfPledge) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	switch method {
	case MethodNameRegister:
		blockParam := blockParamInterface.(*ParamRegister)
		if blockParam.Gid == types.DELEGATE_GID {
			return false
		}
		if ok, _ := regexp.MatchString("^[0-9a-zA-Z_.]{1,40}$", blockParam.Name); !ok {
			return false
		}
		param := new(VariableConditionRegisterOfPledge)
		ABIConsensusGroup.UnpackVariable(param, VariableNameConditionRegisterOfPledge, paramData)
		if block.AccountBlock.Amount.Cmp(param.PledgeAmount) != 0 || block.AccountBlock.TokenId != param.PledgeToken {
			return false
		}
	case MethodNameCancelRegister:
		if block.AccountBlock.Amount.Sign() != 0 ||
			!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
			return false
		}

		param := new(VariableConditionRegisterOfPledge)
		ABIConsensusGroup.UnpackVariable(param, VariableNameConditionRegisterOfPledge, paramData)

		blockParam := blockParamInterface.(*ParamCancelRegister)
		key := GetRegisterKey(blockParam.Name, blockParam.Gid)
		old := new(Registration)
		err := ABIRegister.UnpackVariable(old, VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.ToAddress, key))
		if err != nil || old.PledgeAddr != block.AccountBlock.AccountAddress ||
			!old.IsActive() ||
			old.PledgeHeight+param.PledgeHeight > block.VmContext.CurrentSnapshotBlock().Height {
			return false
		}
	case MethodNameUpdateRegistration:
		if block.AccountBlock.Amount.Sign() != 0 {
			return false
		}
		blockParam := blockParamInterface.(*ParamRegister)
		if blockParam.Gid == types.DELEGATE_GID {
			return false
		}
		old := new(Registration)
		err := ABIRegister.UnpackVariable(
			old,
			VariableNameRegistration,
			block.VmContext.GetStorage(&AddressRegister, GetRegisterKey(blockParam.Name, blockParam.Gid)))
		if err != nil ||
			old.PledgeAddr != block.AccountBlock.AccountAddress ||
			!old.IsActive() ||
			old.NodeAddr == blockParam.NodeAddr {
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
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return false
	}
	return true
}

type voteConditionOfKeepToken struct{}

func (c voteConditionOfKeepToken) checkParam(param []byte, db vmctxt_interface.VmDatabase) bool {
	v := new(VariableConditionVoteOfKeepToken)
	err := ABIConsensusGroup.UnpackVariable(v, VariableNameConditionVoteOfKeepToken, param)
	if err != nil || GetTokenById(db, v.KeepToken) == nil || v.KeepAmount.Sign() == 0 {
		return false
	}
	return true
}
func (c voteConditionOfKeepToken) checkData(paramData []byte, block *vm_context.VmAccountBlock, blockParamInterface interface{}, method string) bool {
	if block.AccountBlock.Amount.Sign() != 0 ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return false
	}
	param := new(VariableConditionVoteOfKeepToken)
	ABIConsensusGroup.UnpackVariable(param, VariableNameConditionVoteOfKeepToken, paramData)
	if block.VmContext.GetBalance(&block.AccountBlock.AccountAddress, &param.KeepToken).Cmp(param.KeepAmount) < 0 {
		return false
	}
	return true
}
