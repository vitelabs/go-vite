package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"regexp"
	"runtime/debug"
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
func (p *MethodRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (p *MethodRegister) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	// Check param by group info
	param := new(abi.ParamRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameRegister, sendBlock.Data)
	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
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
		drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, vm.GlobalStatus().SnapshotBlock())
		util.DealWithErr(err)
		if !drained {
			return nil, util.ErrRewardIsNotDrained
		}
		hisAddrList = old.HisAddrList
	}

	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	v, err := db.GetValue(hisNameKey)
	util.DealWithErr(err)
	err = abi.ABIConsensusGroup.UnpackVariable(hisName, abi.VariableNameHisName, v)
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
func (p *MethodCancelRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
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
func (p *MethodCancelRegister) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamCancelRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameCancelRegister, sendBlock.Data)
	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
	old, err := abi.GetRegistration(db, param.Gid, param.Name)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress || old.WithdrawHeight > snapshotBlock.Height {
		return nil, util.ErrInvalidMethodParam
	}

	rewardTime := old.RewardTime
	cancelTime := snapshotBlock.Timestamp.Unix()
	util.DealWithErr(err)
	drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, vm.GlobalStatus().SnapshotBlock())
	util.DealWithErr(err)
	if drained {
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
		return []*ledger.AccountBlock{
			{
				AccountAddress: block.AccountAddress,
				ToAddress:      sendBlock.AccountAddress,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         old.Amount,
				TokenId:        ledger.ViteTokenId,
				Data:           []byte{},
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
func (p *MethodReward) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
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
func (p *MethodReward) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamReward)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameReward, sendBlock.Data)
	old, err := abi.GetRegistration(db, param.Gid, param.Name)
	util.DealWithErr(err)
	if old == nil || sendBlock.AccountAddress != old.PledgeAddr || old.RewardTime == -1 {
		return nil, util.ErrInvalidMethodParam
	}
	_, endTime, reward, drained, err := CalcReward(vm.ConsensusReader(), db, old, vm.GlobalStatus().SnapshotBlock())
	if err != nil {
		return nil, err
	}
	if drained {
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

		if reward != nil && reward.TotalReward.Sign() > 0 {
			// send reward by issue vite token
			issueData, _ := abi.ABIMintage.PackMethod(abi.MethodNameIssue, ledger.ViteTokenId, reward, param.BeneficialAddr)
			return []*ledger.AccountBlock{
				{
					AccountAddress: block.AccountAddress,
					ToAddress:      types.AddressMintage,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         big.NewInt(0),
					TokenId:        ledger.ViteTokenId,
					Data:           issueData,
				},
			}, nil
		}
	}
	return nil, nil
}

func checkRewardDrained(reader util.ConsensusReader, db vm_db.VmDb, old *types.Registration, current *ledger.SnapshotBlock) (bool, error) {
	_, _, reward, drained, err := CalcReward(reader, db, old, current)
	if err != nil {
		return false, err
	}
	if drained && (reward == nil || reward.TotalReward.Sign() == 0) {
		return drained, nil
	}
	return false, nil
}

type Reward struct {
	VoteReward  *big.Int
	BlockReward *big.Int
	TotalReward *big.Int
}

func newZeroReward() *Reward {
	return &Reward{big.NewInt(0), big.NewInt(0), big.NewInt(0)}
}

func (r *Reward) Add(a *Reward) {
	r.VoteReward.Add(r.VoteReward, a.VoteReward)
	r.BlockReward.Add(r.BlockReward, a.BlockReward)
	r.TotalReward.Add(r.TotalReward, a.TotalReward)
}

func CalcReward(reader util.ConsensusReader, db vm_db.VmDb, old *types.Registration, current *ledger.SnapshotBlock) (startTime int64, endTime int64, reward *Reward, drained bool, err error) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			err = util.ErrChainForked
		}
	}()
	genesisTime := db.GetGenesisSnapshotBlock().Timestamp.Unix()
	pledgeAmount, err := getSnapshotGroupPledgeAmount(db)
	if err != nil {
		return 0, 0, nil, false, err
	}
	return calcReward(old, genesisTime, pledgeAmount, current, reader)
}

func calcReward(old *types.Registration, genesisTime int64, pledgeAmount *big.Int, current *ledger.SnapshotBlock, reader util.ConsensusReader) (int64, int64, *Reward, bool, error) {
	var startIndex, endIndex uint64
	var startTime, endTime int64
	drained := false
	startIndex, startTime, drained = reader.GetIndexByStartTime(old.RewardTime, genesisTime)
	if drained {
		return 0, 0, nil, true, nil
	}
	var withinOneDayFlag bool
	timeLimit := current.Timestamp.Unix() - nodeConfig.params.GetRewardTimeLimit
	if !old.IsActive() && old.CancelTime <= timeLimit {
		drained = true
		endIndex, endTime, withinOneDayFlag = reader.GetIndexByEndTime(old.CancelTime, genesisTime)
	} else {
		endIndex, endTime, withinOneDayFlag = reader.GetIndexByEndTime(timeLimit, genesisTime)
	}
	if withinOneDayFlag || startIndex > endIndex {
		return startTime, endTime, newZeroReward(), drained, nil
	}
	details, err := reader.GetConsensusDetailByDay(startIndex, endIndex)
	if err != nil {
		return 0, 0, nil, false, err
	}
	reward := newZeroReward()
	for _, detail := range details {
		reward.Add(calcRewardByDayDetail(detail, old.Name, pledgeAmount))
	}
	return startTime, endTime, reward, drained, nil
}

func getSnapshotGroupPledgeAmount(db vm_db.VmDb) (*big.Int, error) {
	group, err := abi.GetConsensusGroup(db, types.SNAPSHOT_GID)
	if err != nil {
		return nil, err
	}
	pledgeParam, err := abi.GetRegisterOfPledgeInfo(group.RegisterConditionParam)
	if err != nil {
		return nil, err
	}
	return pledgeParam.PledgeAmount, nil
}

func CalcRewardByDay(db vm_db.VmDb, reader util.ConsensusReader, timestamp int64) (m map[string]*Reward, err error) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			err = util.ErrChainForked
		}
	}()
	genesisTime := db.GetGenesisSnapshotBlock().Timestamp.Unix()
	pledgeAmount, err := getSnapshotGroupPledgeAmount(db)
	if err != nil {
		return nil, err
	}
	return calcRewardByDay(reader, genesisTime, timestamp, pledgeAmount)
}

func calcRewardByDay(reader util.ConsensusReader, genesisTime int64, timestamp int64, pledgeAmount *big.Int) (m map[string]*Reward, err error) {
	index := reader.GetIndexByTime(timestamp, genesisTime)
	detailList, err := reader.GetConsensusDetailByDay(index, index)
	if err != nil {
		return nil, err
	}
	if len(detailList) == 0 {
		return nil, nil
	}
	rewardMap := make(map[string]*Reward, len(detailList[0].Stats))
	for name, _ := range detailList[0].Stats {
		rewardMap[name] = calcRewardByDayDetail(detailList[0], name, pledgeAmount)
	}
	return rewardMap, nil
}

func calcRewardByDayDetail(detail *core.DayStats, name string, pledgeAmount *big.Int) *Reward {
	selfDetail, ok := detail.Stats[name]
	if !ok {
		return newZeroReward()
	}
	reward := &Reward{}

	// reward = 0.5 * rewardPerBlock * totalBlockNum * (selfProducedBlockNum / expectedBlockNum) * (selfVoteCount + pledgeAmount) / (selfVoteCount + totalPledgeAmount)
	// 			+ 0.5 * rewardPerBlock * selfProducedBlockNum
	tmp1 := new(big.Int)
	tmp2 := new(big.Int)
	tmp1.Set(selfDetail.VoteCnt)
	tmp1.Add(tmp1, pledgeAmount)
	tmp2.SetUint64(detail.BlockTotal)
	tmp1.Mul(tmp1, tmp2)
	tmp1.Mul(tmp1, helper.Big50)
	tmp1.Mul(tmp1, rewardPerBlock)
	tmp2.SetUint64(selfDetail.BlockNum)
	tmp1.Mul(tmp1, tmp2)

	tmp2.SetInt64(int64(len(detail.Stats)))
	tmp2.Mul(tmp2, pledgeAmount)
	tmp2.Add(tmp2, detail.VoteSum)
	tmp1.Quo(tmp1, tmp2)

	tmp2.SetUint64(selfDetail.ExceptedBlockNum)
	tmp1.Quo(tmp1, tmp2)
	tmp1.Quo(tmp1, helper.Big100)
	reward.VoteReward = tmp1

	tmp2.SetUint64(selfDetail.BlockNum)
	tmp2.Mul(tmp2, helper.Big50)
	tmp2.Mul(tmp2, rewardPerBlock)
	tmp2.Quo(tmp2, helper.Big100)
	reward.BlockReward = tmp2

	reward.TotalReward = new(big.Int).Add(tmp1, tmp2)
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
func (p *MethodUpdateRegistration) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
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
func (p *MethodUpdateRegistration) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
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
	v, err := db.GetValue(hisNameKey)
	util.DealWithErr(err)
	err = abi.ABIConsensusGroup.UnpackVariable(hisName, abi.VariableNameHisName, v)
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
