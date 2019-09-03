package contracts

import (
	"github.com/vitelabs/go-vite/common/fork"
	"math/big"
	"regexp"
	"runtime/debug"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
)

type MethodRegister struct {
	MethodName string
}

func (p *MethodRegister) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodRegister) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodRegister) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.RegisterGas, nil
}
func (p *MethodRegister) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *MethodRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamRegister)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(p.MethodName, param.Gid, param.SbpName, param.BlockProducingAddress)
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
	abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, sendBlock.Data)
	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
	groupInfo, err := abi.GetConsensusGroup(db, param.Gid)
	util.DealWithErr(err)
	if groupInfo == nil {
		return nil, util.ErrInvalidMethodParam
	}
	pledgeParam, _ := abi.GetRegisterOfPledgeInfo(groupInfo.RegisterConditionParam)
	sb, err := db.LatestSnapshotBlock()
	if isLeafFork := fork.IsLeafFork(sb.Height); (!isLeafFork && sendBlock.Amount.Cmp(SbpStakeAmountPreMainnet) != 0) ||
		(isLeafFork && sendBlock.Amount.Cmp(SbpStakeAmountMainnet) != 0) ||
		sendBlock.TokenId != pledgeParam.PledgeToken {
		return nil, util.ErrInvalidMethodParam
	}

	var rewardTime = int64(0)
	if util.IsSnapshotGid(param.Gid) {
		rewardTime = snapshotBlock.Timestamp.Unix()
	}

	// Check registration owner
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	var hisAddrList []types.Address
	if old != nil {
		if old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
			return nil, util.ErrInvalidMethodParam
		}
		// old is not active, check old reward drained
		drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, snapshotBlock)
		util.DealWithErr(err)
		if !drained {
			return nil, util.ErrRewardIsNotDrained
		}
		hisAddrList = old.HisAddrList
	}

	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.BlockProducingAddress, param.Gid)
	hisName := new(string)
	v := util.GetValue(db, hisNameKey)
	if len(v) == 0 {
		// hisName not exist, update hisName
		hisAddrList = append(hisAddrList, param.BlockProducingAddress)
		hisNameData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, param.SbpName)
		util.SetValue(db, hisNameKey, hisNameData)
	} else {
		err = abi.ABIConsensusGroup.UnpackVariable(hisName, abi.VariableNameHisName, v)
		if err != nil || (err == nil && *hisName != param.SbpName) {
			return nil, util.ErrInvalidMethodParam
		}
	}

	registerInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameRegistration,
		param.SbpName,
		param.BlockProducingAddress,
		sendBlock.AccountAddress,
		sendBlock.Amount,
		snapshotBlock.Height+pledgeParam.PledgeHeight,
		rewardTime,
		int64(0),
		hisAddrList)
	util.SetValue(db, abi.GetRegisterKey(param.SbpName, param.Gid), registerInfo)
	return nil, nil
}

type MethodCancelRegister struct {
	MethodName string
}

func (p *MethodCancelRegister) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodCancelRegister) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodCancelRegister) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.CancelRegisterGas, nil
}
func (p *MethodCancelRegister) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *MethodCancelRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelRegister)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(p.MethodName, param.Gid, param.SbpName)
	return nil
}
func (p *MethodCancelRegister) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamCancelRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, sendBlock.Data)
	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress || old.WithdrawHeight > snapshotBlock.Height {
		return nil, util.ErrInvalidMethodParam
	}

	rewardTime := old.RewardTime
	cancelTime := snapshotBlock.Timestamp.Unix()
	drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, snapshotBlock)
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
	util.SetValue(db, abi.GetRegisterKey(param.SbpName, param.Gid), registerInfo)
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
	MethodName string
}

func (p *MethodReward) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodReward) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodReward) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.RewardGas, nil
}
func (p *MethodReward) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

// get reward of generating snapshot block
func (p *MethodReward) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamReward)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !util.IsSnapshotGid(param.Gid) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(p.MethodName, param.Gid, param.SbpName, param.ReceiveAddress)
	return nil
}
func (p *MethodReward) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamReward)
	abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, sendBlock.Data)
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	if old == nil || sendBlock.AccountAddress != old.PledgeAddr || old.RewardTime == -1 {
		return nil, util.ErrInvalidMethodParam
	}
	_, endTime, reward, drained, err := CalcReward(vm.ConsensusReader(), db, old, vm.GlobalStatus().SnapshotBlock())
	util.DealWithErr(err)
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
		util.SetValue(db, abi.GetRegisterKey(param.SbpName, param.Gid), registerInfo)

		if reward != nil && reward.TotalReward.Sign() > 0 {
			// send reward by issue vite token
			var methodName string
			if !util.CheckFork(db, fork.IsLeafFork) {
				methodName = abi.MethodNameIssue
			} else {
				methodName = abi.MethodNameIssueV2
			}
			issueData, _ := abi.ABIMintage.PackMethod(methodName, ledger.ViteTokenId, reward.TotalReward, param.ReceiveAddress)
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
	return RewardDrained(reward, drained), nil
}

func RewardDrained(reward *Reward, drained bool) bool {
	if drained && (reward == nil || reward.TotalReward.Sign() == 0) {
		return true
	}
	return false
}

type Reward struct {
	VoteReward       *big.Int
	BlockReward      *big.Int
	TotalReward      *big.Int
	BlockNum         uint64
	ExpectedBlockNum uint64
}

func newZeroReward() *Reward {
	return &Reward{big.NewInt(0), big.NewInt(0), big.NewInt(0), 0, 0}
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
	return calcReward(old, genesisTime, db, current, reader)
}

func calcReward(old *types.Registration, genesisTime int64, db vm_db.VmDb, current *ledger.SnapshotBlock, reader util.ConsensusReader) (int64, int64, *Reward, bool, error) {
	var startIndex, endIndex uint64
	var startTime, endTime int64
	drained := false
	startIndex, startTime, drained = reader.GetIndexByStartTime(old.RewardTime, genesisTime)
	if drained {
		return 0, 0, nil, true, nil
	}
	var withinOneDayFlag bool
	timeLimit := getRewardTimeLimit(current)
	if !old.IsActive() {
		endIndex, endTime, withinOneDayFlag = reader.GetIndexByEndTime(old.CancelTime, genesisTime)
		if endTime <= timeLimit {
			drained = true
		} else {
			endIndex, endTime, withinOneDayFlag = reader.GetIndexByEndTime(timeLimit, genesisTime)
		}
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
	forkIndex := uint64(0)
	for _, detail := range details {
		var pledgeAmount *big.Int
		pledgeAmount, forkIndex, err = getSnapshotGroupPledgeAmount(db, reader, genesisTime, detail.Index, forkIndex)
		if err != nil {
			return 0, 0, nil, false, err
		}
		reward.Add(calcRewardByDayDetail(detail, old.Name, pledgeAmount))
	}
	return startTime, endTime, reward, drained, nil
}

func getRewardTimeLimit(current *ledger.SnapshotBlock) int64 {
	return current.Timestamp.Unix() - GetRewardTimeLimit
}

func getSnapshotGroupPledgeAmount(db vm_db.VmDb, reader util.ConsensusReader, genesisTime int64, index uint64, forkIndex uint64) (*big.Int, uint64, error) {
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, forkIndex, err
	}
	if !fork.IsLeafFork(sb.Height) {
		return SbpStakeAmountPreMainnet, 0, nil
	}
	if forkIndex == 0 {
		forkHeight := fork.GetLeafFork()
		forkSb, err := db.GetSnapshotBlockByHeight(forkHeight)
		if err != nil {
			return nil, forkIndex, err
		}
		forkIndex = reader.GetIndexByTime(forkSb.Timestamp.Unix(), genesisTime)
	}
	if index <= forkIndex {
		return SbpStakeAmountPreMainnet, forkIndex, nil
	}
	return SbpStakeAmountMainnet, forkIndex, nil
}

func CalcRewardByDay(db vm_db.VmDb, reader util.ConsensusReader, timestamp int64) (m map[string]*Reward, index uint64, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			debug.PrintStack()
			err = util.ErrChainForked
		}
	}()
	genesisTime := db.GetGenesisSnapshotBlock().Timestamp.Unix()
	if timestamp < genesisTime {
		return nil, 0, util.ErrInvalidMethodParam
	}
	index = reader.GetIndexByTime(timestamp, genesisTime)
	endTime := reader.GetEndTimeByIndex(index)
	current, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, 0, err
	}
	timeLimit := getRewardTimeLimit(current)
	if endTime > timeLimit {
		return nil, 0, util.ErrRewardNotDue
	}
	pledgeAmount, _, err := getSnapshotGroupPledgeAmount(db, reader, genesisTime, index, 0)
	if err != nil {
		return nil, 0, err
	}
	m, err = calcRewardByDay(reader, index, pledgeAmount)
	if err != nil {
		return nil, 0, err
	}
	return m, index, nil
}

func CalcRewardByDayIndex(db vm_db.VmDb, reader util.ConsensusReader, index uint64) (m map[string]*Reward, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			debug.PrintStack()
			err = util.ErrChainForked
		}
	}()
	endTime := reader.GetEndTimeByIndex(index)
	current, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	timeLimit := getRewardTimeLimit(current)
	if endTime > timeLimit {
		return nil, util.ErrRewardNotDue
	}
	genesisTime := db.GetGenesisSnapshotBlock().Timestamp.Unix()
	pledgeAmount, _, err := getSnapshotGroupPledgeAmount(db, reader, genesisTime, index, 0)
	if err != nil {
		return nil, err
	}
	return calcRewardByDay(reader, index, pledgeAmount)
}

func calcRewardByDay(reader util.ConsensusReader, index uint64, pledgeAmount *big.Int) (m map[string]*Reward, err error) {
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
	if !ok || selfDetail.ExceptedBlockNum == 0 {
		return newZeroReward()
	}
	reward := &Reward{}

	// reward = 0.5 * rewardPerBlock * totalBlockNum * (selfProducedBlockNum / expectedBlockNum) * (selfVoteCount + pledgeAmount) / (selfVoteCount + totalPledgeAmount)
	// 			+ 0.5 * rewardPerBlock * selfProducedBlockNum
	tmp1 := new(big.Int)
	tmp2 := new(big.Int)
	tmp1.Set(selfDetail.VoteCnt.Int)
	tmp1.Add(tmp1, pledgeAmount)
	tmp2.SetUint64(detail.BlockTotal)
	tmp1.Mul(tmp1, tmp2)
	tmp1.Mul(tmp1, helper.Big50)
	tmp1.Mul(tmp1, rewardPerBlock)
	tmp2.SetUint64(selfDetail.BlockNum)
	tmp1.Mul(tmp1, tmp2)
	tmp2.SetInt64(int64(len(detail.Stats)))
	tmp2.Mul(tmp2, pledgeAmount)
	tmp2.Add(tmp2, detail.VoteSum.Int)
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
	reward.BlockNum = selfDetail.BlockNum
	reward.ExpectedBlockNum = selfDetail.ExceptedBlockNum
	return reward
}

type MethodUpdateRegistration struct {
	MethodName string
}

func (p *MethodUpdateRegistration) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodUpdateRegistration) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodUpdateRegistration) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return gasTable.UpdateRegistrationGas, nil
}
func (p *MethodUpdateRegistration) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return 0
}

// update registration info
func (p *MethodUpdateRegistration) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamRegister)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(p.MethodName, param.Gid, param.SbpName, param.BlockProducingAddress)
	return nil
}
func (p *MethodUpdateRegistration) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamRegister)
	abi.ABIConsensusGroup.UnpackMethod(param, p.MethodName, sendBlock.Data)
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() ||
		old.PledgeAddr != sendBlock.AccountAddress ||
		old.NodeAddr == param.BlockProducingAddress {
		return nil, util.ErrInvalidMethodParam
	}
	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.BlockProducingAddress, param.Gid)
	hisName := new(string)
	v := util.GetValue(db, hisNameKey)
	if len(v) == 0 {
		// hisName not exist, update hisName
		old.HisAddrList = append(old.HisAddrList, param.BlockProducingAddress)
		hisNameData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, param.SbpName)
		util.SetValue(db, hisNameKey, hisNameData)
	} else {
		err = abi.ABIConsensusGroup.UnpackVariable(hisName, abi.VariableNameHisName, v)
		if err != nil || (err == nil && *hisName != param.SbpName) {
			return nil, util.ErrInvalidMethodParam
		}
	}
	registerInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameRegistration,
		old.Name,
		param.BlockProducingAddress,
		old.PledgeAddr,
		old.Amount,
		old.WithdrawHeight,
		old.RewardTime,
		old.CancelTime,
		old.HisAddrList)
	util.SetValue(db, abi.GetRegisterKey(param.SbpName, param.Gid), registerInfo)
	return nil, nil
}
