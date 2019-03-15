package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"regexp"
	"runtime/debug"
	"time"
)

type MethodRegister struct {
}

func (p *MethodRegister) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodRegister) GetRefundData() []byte {
	return []byte{1}
}
func (p *MethodRegister) GetSendQuota(data []byte) (uint64, error) {
	return RegisterGas, nil
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *MethodRegister) DoSend(db vm_db.VMDB, block *ledger.AccountBlock) error {
	if !util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamRegister)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameRegister, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.Name) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameRegister, param.Gid, param.Name, param.NodeAddr)
	return nil
}

func checkRegisterAndVoteParam(gid types.Gid, name string) bool {
	if util.IsDelegateGid(gid) ||
		len(name) == 0 ||
		len(name) > registrationNameLengthMax {
		return false
	}
	if ok, _ := regexp.MatchString("^([0-9a-zA-Z_.]+[ ]?)*[0-9a-zA-Z_.]$", name); !ok {
		return false
	}
	return true
}

func (p *MethodRegister) DoReceive(db vm_db.VMDB, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	// Check param by group info
	param := new(abi.ParamRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameRegister, sendBlock.Data)
	snapshotBlock := globalStatus.SnapshotBlock
	groupInfo, err := abi.GetConsensusGroup(db, param.Gid)
	util.DealWithErr(err)
	if groupInfo == nil {
		return nil, util.ErrInvalidMethodParam
	}
	pledgeParam, _ := abi.GetRegisterOfPledgeInfo(groupInfo.RegisterConditionParam)
	if sendBlock.Amount.Cmp(pledgeParam.PledgeAmount) != 0 || sendBlock.TokenId != pledgeParam.PledgeToken {
		return nil, util.ErrInvalidMethodParam
	}

	var rewardTime = int64(0)
	if util.IsSnapshotGid(param.Gid) {
		rewardTime = snapshotBlock.Timestamp.Unix()
	}

	// Check registration owner
	old, err := abi.GetRegistration(db, param.Gid, param.Name)
	util.DealWithErr(err)
	var hisAddrList []types.Address
	if old != nil {
		if old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
			return nil, util.ErrInvalidMethodParam
		}
		// old is not active, check old reward drained
		if !checkRewardDrained(db, groupInfo, old.RewardTime, old.CancelTime) {
			return nil, errors.New("reward is not drained")
		}
		hisAddrList = old.HisAddrList
	}

	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	err = abi.ABIConsensusGroup.UnpackVariable(hisName, abi.VariableNameHisName, db.GetValue(hisNameKey))
	if err == nil && *hisName != param.Name {
		return nil, util.ErrInvalidMethodParam
	}
	if err != nil {
		// hisName not exist, update hisName
		hisAddrList = append(hisAddrList, param.NodeAddr)
		hisNameData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, param.Name)
		db.SetValue(hisNameKey, hisNameData)
	}

	registerInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameRegistration,
		param.Name,
		param.NodeAddr,
		sendBlock.AccountAddress,
		sendBlock.Amount,
		snapshotBlock.Height+pledgeParam.PledgeHeight,
		rewardTime,
		int64(0),
		hisAddrList)
	db.SetValue(abi.GetRegisterKey(param.Name, param.Gid), registerInfo)
	return nil, nil
}

type MethodCancelRegister struct {
}

func (p *MethodCancelRegister) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodCancelRegister) GetRefundData() []byte {
	return []byte{2}
}
func (p *MethodCancelRegister) GetSendQuota(data []byte) (uint64, error) {
	return CancelRegisterGas, nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *MethodCancelRegister) DoSend(db vm_db.VMDB, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelRegister)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameCancelRegister, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.Name) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelRegister, param.Gid, param.Name)
	return nil
}
func (p *MethodCancelRegister) DoReceive(db vm_db.VMDB, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	param := new(abi.ParamCancelRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameCancelRegister, sendBlock.Data)
	snapshotBlock := globalStatus.SnapshotBlock
	old, err := abi.GetRegistration(db, param.Gid, param.Name)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress || old.WithdrawHeight > snapshotBlock.Height {
		return nil, util.ErrInvalidMethodParam
	}

	rewardTime := old.RewardTime
	cancelTime := snapshotBlock.Timestamp.Unix()
	groupInfo, err := abi.GetConsensusGroup(db, param.Gid)
	util.DealWithErr(err)
	if groupInfo == nil {
		return nil, util.ErrInvalidMethodParam
	}
	if checkRewardDrained(db, groupInfo, old.RewardTime, cancelTime) {
		rewardTime = -1
	}

	registerInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameRegistration,
		old.Name,
		old.NodeAddr,
		old.PledgeAddr,
		helper.Big0,
		uint64(0),
		rewardTime,
		cancelTime,
		old.HisAddrList)
	db.SetValue(abi.GetRegisterKey(param.Name, param.Gid), registerInfo)
	if old.Amount.Sign() > 0 {
		return []*SendBlock{
			{
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				old.Amount,
				ledger.ViteTokenId,
				[]byte{},
			},
		}, nil
	}
	return nil, nil
}

type MethodReward struct {
}

func (p *MethodReward) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodReward) GetRefundData() []byte {
	return []byte{3}
}
func (p *MethodReward) GetSendQuota(data []byte) (uint64, error) {
	return RewardGas, nil
}

// get reward of generating snapshot block
func (p *MethodReward) DoSend(db vm_db.VMDB, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamReward)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameReward, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !util.IsSnapshotGid(param.Gid) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameReward, param.Gid, param.Name, param.BeneficialAddr)
	return nil
}
func (p *MethodReward) DoReceive(db vm_db.VMDB, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	param := new(abi.ParamReward)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameReward, sendBlock.Data)
	old, err := abi.GetRegistration(db, param.Gid, param.Name)
	util.DealWithErr(err)
	if old == nil || sendBlock.AccountAddress != old.PledgeAddr || old.RewardTime == -1 {
		return nil, util.ErrInvalidMethodParam
	}
	startTime, endTime, reward, err := CalcReward(db, old, param.Gid, globalStatus.SnapshotBlock)
	if err != nil {
		return nil, err
	}
	if startTime == endTime {
		endTime = -1
	}
	if endTime != old.RewardTime {
		registerInfo, _ := abi.ABIConsensusGroup.PackVariable(
			abi.VariableNameRegistration,
			old.Name,
			old.NodeAddr,
			old.PledgeAddr,
			old.Amount,
			old.WithdrawHeight,
			endTime,
			old.CancelTime,
			old.HisAddrList)
		db.SetValue(abi.GetRegisterKey(param.Name, param.Gid), registerInfo)

		if reward != nil && reward.Sign() > 0 {
			// send reward by issue vite token
			issueData, _ := abi.ABIMintage.PackMethod(abi.MethodNameIssue, ledger.ViteTokenId, reward, param.BeneficialAddr)
			return []*SendBlock{
				{
					types.AddressMintage,
					ledger.BlockTypeSendCall,
					big.NewInt(0),
					ledger.ViteTokenId,
					issueData,
				},
			}, nil
		}
	}
	return nil, nil
}

func checkRewardDrained(db vm_db.VMDB, groupInfo *types.ConsensusGroupInfo, rewardTime, cancelTime int64) bool {
	if rewardTime == -1 {
		return true
	}
	// old is not active, check old reward drained
	reader := newConsensusReader(db.GetGenesisSnapshotBlock().Timestamp, groupInfo)
	if reader.timeToRewardEndDayTime(cancelTime) == reader.timeToRewardEndDayTime(rewardTime) {
		return true
	}
	return false
}

func CalcReward(db vm_db.VMDB, old *types.Registration, gid types.Gid, current *ledger.SnapshotBlock) (startTime int64, endTime int64, reward *big.Int, err error) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			err = errors.New("calc reward panic")
		}
	}()
	groupInfo, err := abi.GetConsensusGroup(db, gid)
	util.DealWithErr(err)
	reader := newConsensusReader(db.GetGenesisSnapshotBlock().Timestamp, groupInfo)
	return calcReward(old, current, reader, groupInfo)
}

func calcReward(old *types.Registration, current *ledger.SnapshotBlock, reader *consensusReader, groupInfo *types.ConsensusGroupInfo) (startTime int64, endTime int64, reward *big.Int, err error) {
	if old.RewardTime < reader.genesisTime {
		startTime = reader.timeToRewardStartDayTime(reader.genesisTime)
	} else {
		startTime = reader.timeToRewardStartDayTime(old.RewardTime)
	}
	if !old.IsActive() {
		endTime = reader.timeToRewardEndDayTime(old.CancelTime)
	} else {
		endTime = reader.timeToRewardEndDayTime(current.Timestamp.Unix() - nodeConfig.params.GetRewardTimeLimit)
	}
	if startTime == endTime {
		// reward drained
		return startTime, endTime, nil, nil
	}
	indexInDay := reader.getIndexInDay()
	startIndex := reader.timeToPeriodIndex(time.Unix(startTime, 0))
	endIndex := reader.timeToPeriodIndex(time.Unix(endTime, 0))
	reward = big.NewInt(0)
	tmp1 := big.NewInt(0)
	tmp2 := big.NewInt(0)
	pledgeParam, _ := abi.GetRegisterOfPledgeInfo(groupInfo.RegisterConditionParam)
	for startIndex < endIndex {
		detailMap, summary := reader.getConsensusDetailByDay(startIndex, endIndex)
		reward.Add(reward, calcRewardByDay(detailMap, summary, old.Name, pledgeParam.PledgeAmount, tmp1, tmp2))
		startIndex = startIndex + indexInDay
	}
	return startTime, endTime, reward, nil
}

func calcRewardByDay(detailMap map[string]*consensusDetail, summary *consensusDetail, name string, pledgeAmount *big.Int, tmp1, tmp2 *big.Int) *big.Int {
	reward := big.NewInt(0)
	selfDetail, ok := detailMap[name]
	if !ok {
		return reward
	}

	// reward = 0.5 * rewardPerBlock * totalBlockNum * (selfProducedBlockNum / expectedBlockNum) * (selfVoteCount * pledgeAmount) / (selfVoteCount + totalPledgeAmount)
	// 			+ 0.5 * rewardPerBlock * selfProducedBlockNum
	tmp1.Set(selfDetail.voteCount)
	tmp1.Add(tmp1, pledgeAmount)
	tmp2.SetUint64(summary.blockNum)
	tmp1.Mul(tmp1, tmp2)
	tmp1.Mul(tmp1, helper.Big50)
	tmp1.Mul(tmp1, rewardPerBlock)
	tmp1.Mul(tmp1, tmp2)

	tmp2.SetInt64(int64(len(detailMap)))
	tmp2.Mul(tmp2, pledgeAmount)
	tmp2.Add(tmp2, selfDetail.voteCount)
	tmp1.Quo(tmp1, tmp2)
	tmp2.SetUint64(selfDetail.expectedBlockNum)
	tmp1.Quo(tmp1, tmp2)

	tmp2.SetUint64(selfDetail.blockNum)
	tmp2.Mul(tmp2, helper.Big50)
	tmp2.Mul(tmp2, rewardPerBlock)

	tmp1.Add(tmp1, tmp2)
	tmp1.Quo(tmp1, helper.Big100)

	reward.Set(tmp1)
	return reward
}

type MethodUpdateRegistration struct {
}

func (p *MethodUpdateRegistration) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodUpdateRegistration) GetRefundData() []byte {
	return []byte{4}
}
func (p *MethodUpdateRegistration) GetSendQuota(data []byte) (uint64, error) {
	return UpdateRegistrationGas, nil
}

// update registration info
func (p *MethodUpdateRegistration) DoSend(db vm_db.VMDB, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamRegister)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameUpdateRegistration, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.Name) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameUpdateRegistration, param.Gid, param.Name, param.NodeAddr)
	return nil
}
func (p *MethodUpdateRegistration) DoReceive(db vm_db.VMDB, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	param := new(abi.ParamRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameUpdateRegistration, sendBlock.Data)
	old, err := abi.GetRegistration(db, param.Gid, param.Name)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() ||
		old.PledgeAddr != sendBlock.AccountAddress ||
		old.NodeAddr == param.NodeAddr {
		return nil, util.ErrInvalidMethodParam
	}
	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	err = abi.ABIConsensusGroup.UnpackVariable(hisName, abi.VariableNameHisName, db.GetValue(hisNameKey))
	if err == nil && *hisName != param.Name {
		return nil, util.ErrInvalidMethodParam
	}
	if err != nil {
		// hisName not exist, update hisName
		old.HisAddrList = append(old.HisAddrList, param.NodeAddr)
		hisNameData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, param.Name)
		db.SetValue(hisNameKey, hisNameData)
	}
	registerInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameRegistration,
		old.Name,
		param.NodeAddr,
		old.PledgeAddr,
		old.Amount,
		old.WithdrawHeight,
		old.RewardTime,
		old.CancelTime,
		old.HisAddrList)
	db.SetValue(abi.GetRegisterKey(param.Name, param.Gid), registerInfo)
	return nil, nil
}
