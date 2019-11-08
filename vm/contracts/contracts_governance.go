package contracts

import (
	"github.com/vitelabs/go-vite/common/fork"
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
	"strings"
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
func (p *MethodRegister) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.RegisterQuota, nil
}
func (p *MethodRegister) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodRegister) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(abi.ParamRegister)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameRegisterV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameRegisterV3 {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName, param.BlockProducingAddress, param.RewardWithdrawAddress)
	} else {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName, param.BlockProducingAddress)
	}
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
	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
	if p.MethodName == abi.MethodNameRegisterV3 {
		param.Gid = types.SNAPSHOT_GID
	} else {
		param.RewardWithdrawAddress = sendBlock.AccountAddress
	}

	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
	sb, err := db.LatestSnapshotBlock()
	if isLeafFork := fork.IsLeafFork(sb.Height); (!isLeafFork && sendBlock.Amount.Cmp(SbpStakeAmountPreMainnet) != 0) ||
		(isLeafFork && sendBlock.Amount.Cmp(SbpStakeAmountMainnet) != 0) ||
		sendBlock.TokenId != ledger.ViteTokenId {
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
	var oldWithdrawRewardAddress *types.Address
	if old != nil {
		if old.IsActive() || old.StakeAddress != sendBlock.AccountAddress {
			return nil, util.ErrInvalidMethodParam
		}
		// old is not active, check old reward drained
		drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, snapshotBlock)
		util.DealWithErr(err)
		if !drained {
			return nil, util.ErrRewardIsNotDrained
		}
		hisAddrList = old.HisAddrList
		oldWithdrawRewardAddress = &old.RewardWithdrawAddress
	}

	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.BlockProducingAddress, param.Gid)
	hisName := new(string)
	v := util.GetValue(db, hisNameKey)
	if len(v) == 0 {
		// hisName not exist, update hisName
		hisAddrList = append(hisAddrList, param.BlockProducingAddress)
		hisNameData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, param.SbpName)
		util.SetValue(db, hisNameKey, hisNameData)
	} else {
		err = abi.ABIGovernance.UnpackVariable(hisName, abi.VariableNameRegisteredHisName, v)
		if err != nil || (err == nil && *hisName != param.SbpName) {
			return nil, util.ErrInvalidMethodParam
		}
	}

	groupInfo, err := abi.GetConsensusGroup(db, param.Gid)
	util.DealWithErr(err)
	if groupInfo == nil {
		return nil, util.ErrInvalidMethodParam
	}
	stakeParam, _ := abi.GetRegisterStakeParamOfConsensusGroup(groupInfo.RegisterConditionParam)

	var registerInfo []byte
	if fork.IsEarthFork(sb.Height) {
		// save withdraw reward address -> sbp name
		saveWithdrawRewardAddress(db, oldWithdrawRewardAddress, param.RewardWithdrawAddress, sendBlock.AccountAddress, param.SbpName)
		registerInfo, _ = abi.ABIGovernance.PackVariable(
			abi.VariableNameRegistrationInfoV2,
			param.SbpName,
			param.BlockProducingAddress,
			param.RewardWithdrawAddress,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			snapshotBlock.Height+stakeParam.StakeHeight,
			rewardTime,
			int64(0),
			hisAddrList)
	} else {
		registerInfo, _ = abi.ABIGovernance.PackVariable(
			abi.VariableNameRegistrationInfo,
			param.SbpName,
			param.BlockProducingAddress,
			sendBlock.AccountAddress,
			sendBlock.Amount,
			snapshotBlock.Height+stakeParam.StakeHeight,
			rewardTime,
			int64(0),
			hisAddrList)
	}
	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
	return nil, nil
}

func saveWithdrawRewardAddress(db vm_db.VmDb, oldAddr *types.Address, newAddr, owner types.Address, sbpName string) {
	if oldAddr == nil {
		if newAddr == owner {
			return
		}
		addWithdrawRewardAddress(db, newAddr, sbpName)
		return
	}
	if *oldAddr == newAddr {
		return
	}
	if *oldAddr == owner {
		addWithdrawRewardAddress(db, newAddr, sbpName)
		return
	}
	if newAddr == owner {
		deleteWithdrawRewardAddress(db, *oldAddr, sbpName)
		return
	}
	deleteWithdrawRewardAddress(db, *oldAddr, sbpName)
	addWithdrawRewardAddress(db, newAddr, sbpName)
}

func addWithdrawRewardAddress(db vm_db.VmDb, addr types.Address, sbpName string) {
	value := util.GetValue(db, addr.Bytes())
	if len(value) == 0 {
		value = []byte(sbpName)
	} else {
		value = []byte(string(value) + abi.WithdrawRewardAddressSeparation + sbpName)
	}
	util.SetValue(db, addr.Bytes(), value)
}
func deleteWithdrawRewardAddress(db vm_db.VmDb, addr types.Address, sbpName string) {
	value := util.GetValue(db, addr.Bytes())
	valueStr := string(value)
	// equal, prefix, suffix, middle
	if valueStr == sbpName {
		util.SetValue(db, addr.Bytes(), nil)
	} else if strings.HasPrefix(valueStr, sbpName+abi.WithdrawRewardAddressSeparation) {
		util.SetValue(db, addr.Bytes(), []byte(valueStr[len(sbpName)+1:]))
	} else if strings.HasSuffix(valueStr, abi.WithdrawRewardAddressSeparation+sbpName) {
		util.SetValue(db, addr.Bytes(), []byte(valueStr[:len(valueStr)-len(sbpName)-1]))
	} else if startIndex := strings.Index(valueStr, abi.WithdrawRewardAddressSeparation+sbpName+abi.WithdrawRewardAddressSeparation); startIndex > 0 {
		util.SetValue(db, addr.Bytes(), append([]byte(valueStr[:startIndex]+valueStr[startIndex+len(sbpName)+1:])))
	}
}

type MethodRevoke struct {
	MethodName string
}

func (p *MethodRevoke) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodRevoke) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodRevoke) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.RevokeQuota, nil
}
func (p *MethodRevoke) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodRevoke) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelRegister)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameRevokeV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameRevokeV3 {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName)
	} else {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName)
	}
	return nil
}
func (p *MethodRevoke) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamCancelRegister)
	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
	if p.MethodName == abi.MethodNameRevokeV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	snapshotBlock := vm.GlobalStatus().SnapshotBlock()
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() || old.StakeAddress != sendBlock.AccountAddress || old.ExpirationHeight > snapshotBlock.Height {
		return nil, util.ErrInvalidMethodParam
	}

	rewardTime := old.RewardTime
	revokeTime := snapshotBlock.Timestamp.Unix()
	drained, err := checkRewardDrained(vm.ConsensusReader(), db, old, snapshotBlock)
	util.DealWithErr(err)
	if drained {
		rewardTime = -1
	}
	var registerInfo []byte
	if fork.IsEarthFork(snapshotBlock.Height) {
		registerInfo, _ = abi.ABIGovernance.PackVariable(
			abi.VariableNameRegistrationInfoV2,
			old.Name,
			old.BlockProducingAddress,
			old.RewardWithdrawAddress,
			old.StakeAddress,
			helper.Big0,
			uint64(0),
			rewardTime,
			revokeTime,
			old.HisAddrList)
	} else {
		registerInfo, _ = abi.ABIGovernance.PackVariable(
			abi.VariableNameRegistrationInfo,
			old.Name,
			old.BlockProducingAddress,
			old.StakeAddress,
			helper.Big0,
			uint64(0),
			rewardTime,
			revokeTime,
			old.HisAddrList)
	}
	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
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

type MethodWithdrawReward struct {
	MethodName string
}

func (p *MethodWithdrawReward) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodWithdrawReward) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodWithdrawReward) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.WithdrawRewardQuota, nil
}
func (p *MethodWithdrawReward) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodWithdrawReward) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamReward)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameWithdrawRewardV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	if !util.IsSnapshotGid(param.Gid) {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameWithdrawRewardV3 {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName, param.ReceiveAddress)
	} else {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName, param.ReceiveAddress)
	}
	return nil
}
func (p *MethodWithdrawReward) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamReward)
	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
	if p.MethodName == abi.MethodNameWithdrawRewardV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	sb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	if old == nil || old.RewardTime == -1 ||
		(sendBlock.AccountAddress != old.StakeAddress && sendBlock.AccountAddress != old.RewardWithdrawAddress) {
		return nil, util.ErrInvalidMethodParam
	}
	_, endTime, reward, drained, err := CalcReward(vm.ConsensusReader(), db, old, vm.GlobalStatus().SnapshotBlock())
	util.DealWithErr(err)
	if drained {
		endTime = -1
	}
	if endTime != old.RewardTime {
		var registerInfo []byte
		if fork.IsEarthFork(sb.Height) {
			registerInfo, _ = abi.ABIGovernance.PackVariable(
				abi.VariableNameRegistrationInfoV2,
				old.Name,
				old.BlockProducingAddress,
				old.RewardWithdrawAddress,
				old.StakeAddress,
				old.Amount,
				old.ExpirationHeight,
				endTime,
				old.RevokeTime,
				old.HisAddrList)
		} else {
			registerInfo, _ = abi.ABIGovernance.PackVariable(
				abi.VariableNameRegistrationInfo,
				old.Name,
				old.BlockProducingAddress,
				old.StakeAddress,
				old.Amount,
				old.ExpirationHeight,
				endTime,
				old.RevokeTime,
				old.HisAddrList)
		}
		util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)

		if reward != nil && reward.TotalReward.Sign() > 0 {
			// send reward by reIssue vite token
			var methodName string
			if !util.CheckFork(db, fork.IsLeafFork) {
				methodName = abi.MethodNameReIssue
			} else {
				methodName = abi.MethodNameReIssueV2
			}
			reIssueData, _ := abi.ABIAsset.PackMethod(methodName, ledger.ViteTokenId, reward.TotalReward, param.ReceiveAddress)
			return []*ledger.AccountBlock{
				{
					AccountAddress: block.AccountAddress,
					ToAddress:      types.AddressAsset,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         big.NewInt(0),
					TokenId:        ledger.ViteTokenId,
					Data:           reIssueData,
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

// RewardDrained checks whether reward is drained
func RewardDrained(reward *Reward, drained bool) bool {
	if drained && (reward == nil || reward.TotalReward.Sign() == 0) {
		return true
	}
	return false
}

// Reward defines snapshot reward details
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

func (r *Reward) add(a *Reward) {
	r.VoteReward.Add(r.VoteReward, a.VoteReward)
	r.BlockReward.Add(r.BlockReward, a.BlockReward)
	r.TotalReward.Add(r.TotalReward, a.TotalReward)
}

// CalcReward calculates available reward of a sbp
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
		endIndex, endTime, withinOneDayFlag = reader.GetIndexByEndTime(old.RevokeTime, genesisTime)
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
		var stakeAmount *big.Int
		stakeAmount, forkIndex, err = getSnapshotGroupStakeAmount(db, reader, genesisTime, detail.Index, forkIndex)
		if err != nil {
			return 0, 0, nil, false, err
		}
		reward.add(calcRewardByDayDetail(detail, old.Name, stakeAmount))
	}
	return startTime, endTime, reward, drained, nil
}

func getRewardTimeLimit(current *ledger.SnapshotBlock) int64 {
	return current.Timestamp.Unix() - rewardTimeLimit
}

func getSnapshotGroupStakeAmount(db vm_db.VmDb, reader util.ConsensusReader, genesisTime int64, index uint64, forkIndex uint64) (*big.Int, uint64, error) {
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, forkIndex, err
	}
	if !fork.IsLeafFork(sb.Height) {
		return SbpStakeAmountPreMainnet, 0, nil
	}
	if forkIndex == 0 {
		forkSb, err := db.GetSnapshotBlockByHeight(fork.GetLeafForkPoint().Height)
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

// CalcRewardByCycle calculates reward of all sbps in one cycle
func CalcRewardByCycle(db vm_db.VmDb, reader util.ConsensusReader, timestamp int64) (m map[string]*Reward, index uint64, err error) {
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
	stakeAmount, _, err := getSnapshotGroupStakeAmount(db, reader, genesisTime, index, 0)
	if err != nil {
		return nil, 0, err
	}
	m, err = calcRewardByDay(reader, index, stakeAmount)
	if err != nil {
		return nil, 0, err
	}
	return m, index, nil
}

// CalcRewardByIndex calculates reward of all sbps in one cycle
func CalcRewardByIndex(db vm_db.VmDb, reader util.ConsensusReader, index uint64) (m map[string]*Reward, err error) {
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
	stakeAmount, _, err := getSnapshotGroupStakeAmount(db, reader, genesisTime, index, 0)
	if err != nil {
		return nil, err
	}
	return calcRewardByDay(reader, index, stakeAmount)
}

func calcRewardByDay(reader util.ConsensusReader, index uint64, stakeAmount *big.Int) (m map[string]*Reward, err error) {
	detailList, err := reader.GetConsensusDetailByDay(index, index)
	if err != nil {
		return nil, err
	}
	if len(detailList) == 0 {
		return nil, nil
	}
	rewardMap := make(map[string]*Reward, len(detailList[0].Stats))
	for name := range detailList[0].Stats {
		rewardMap[name] = calcRewardByDayDetail(detailList[0], name, stakeAmount)
	}
	return rewardMap, nil
}

func calcRewardByDayDetail(detail *core.DayStats, name string, stakeAmount *big.Int) *Reward {
	selfDetail, ok := detail.Stats[name]
	if !ok || selfDetail.ExceptedBlockNum == 0 {
		return newZeroReward()
	}
	reward := &Reward{}

	// reward = 0.5 * rewardPerBlock * totalBlockNum * (selfProducedBlockNum / expectedBlockNum) * (selfVoteCount + stakeAmount) / (selfVoteCount + totalStakeAmount)
	// 			+ 0.5 * rewardPerBlock * selfProducedBlockNum
	tmp1 := new(big.Int)
	tmp2 := new(big.Int)
	tmp1.Set(selfDetail.VoteCnt.Int)
	tmp1.Add(tmp1, stakeAmount)
	tmp2.SetUint64(detail.BlockTotal)
	tmp1.Mul(tmp1, tmp2)
	tmp1.Mul(tmp1, helper.Big50)
	tmp1.Mul(tmp1, rewardPerBlock)
	tmp2.SetUint64(selfDetail.BlockNum)
	tmp1.Mul(tmp1, tmp2)
	tmp2.SetInt64(int64(len(detail.Stats)))
	tmp2.Mul(tmp2, stakeAmount)
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

type MethodUpdateBlockProducingAddress struct {
	MethodName string
}

func (p *MethodUpdateBlockProducingAddress) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodUpdateBlockProducingAddress) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodUpdateBlockProducingAddress) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.UpdateBlockProducingAddressQuota, nil
}
func (p *MethodUpdateBlockProducingAddress) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodUpdateBlockProducingAddress) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamRegister)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameUpdateBlockProducintAddressV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameUpdateBlockProducintAddressV3 {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName, param.BlockProducingAddress)
	} else {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName, param.BlockProducingAddress)
	}
	return nil
}
func (p *MethodUpdateBlockProducingAddress) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamRegister)
	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
	if p.MethodName == abi.MethodNameUpdateBlockProducintAddressV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() ||
		old.StakeAddress != sendBlock.AccountAddress ||
		old.BlockProducingAddress == param.BlockProducingAddress {
		return nil, util.ErrInvalidMethodParam
	}
	// check node addr belong to one name in a consensus group
	hisNameKey := abi.GetHisNameKey(param.BlockProducingAddress, param.Gid)
	hisName := new(string)
	v := util.GetValue(db, hisNameKey)
	if len(v) == 0 {
		// hisName not exist, update hisName
		old.HisAddrList = append(old.HisAddrList, param.BlockProducingAddress)
		hisNameData, _ := abi.ABIGovernance.PackVariable(abi.VariableNameRegisteredHisName, param.SbpName)
		util.SetValue(db, hisNameKey, hisNameData)
	} else {
		err = abi.ABIGovernance.UnpackVariable(hisName, abi.VariableNameRegisteredHisName, v)
		if err != nil || (err == nil && *hisName != param.SbpName) {
			return nil, util.ErrInvalidMethodParam
		}
	}
	var registerInfo []byte
	if util.CheckFork(db, fork.IsEarthFork) {
		registerInfo, _ = abi.ABIGovernance.PackVariable(
			abi.VariableNameRegistrationInfoV2,
			old.Name,
			param.BlockProducingAddress,
			old.RewardWithdrawAddress,
			old.StakeAddress,
			old.Amount,
			old.ExpirationHeight,
			old.RewardTime,
			old.RevokeTime,
			old.HisAddrList)
	} else {
		registerInfo, _ = abi.ABIGovernance.PackVariable(
			abi.VariableNameRegistrationInfo,
			old.Name,
			param.BlockProducingAddress,
			old.StakeAddress,
			old.Amount,
			old.ExpirationHeight,
			old.RewardTime,
			old.RevokeTime,
			old.HisAddrList)
	}
	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
	return nil, nil
}

type MethodUpdateRewardWithdrawAddress struct {
	MethodName string
}

func (p *MethodUpdateRewardWithdrawAddress) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodUpdateRewardWithdrawAddress) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodUpdateRewardWithdrawAddress) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.UpdateRewardWithdrawAddressQuota, nil
}
func (p *MethodUpdateRewardWithdrawAddress) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodUpdateRewardWithdrawAddress) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamRegister)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	param.Gid = types.SNAPSHOT_GID
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName, param.RewardWithdrawAddress)
	return nil
}
func (p *MethodUpdateRewardWithdrawAddress) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamRegister)
	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
	param.Gid = types.SNAPSHOT_GID
	old, err := abi.GetRegistration(db, param.Gid, param.SbpName)
	util.DealWithErr(err)
	if old == nil || !old.IsActive() ||
		old.StakeAddress != sendBlock.AccountAddress ||
		old.RewardWithdrawAddress == param.RewardWithdrawAddress {
		return nil, util.ErrInvalidMethodParam
	}
	saveWithdrawRewardAddress(db, &old.RewardWithdrawAddress, param.RewardWithdrawAddress, old.StakeAddress, old.Name)
	registerInfo, _ := abi.ABIGovernance.PackVariable(
		abi.VariableNameRegistrationInfoV2,
		old.Name,
		old.BlockProducingAddress,
		param.RewardWithdrawAddress,
		old.StakeAddress,
		old.Amount,
		old.ExpirationHeight,
		old.RewardTime,
		old.RevokeTime,
		old.HisAddrList)
	util.SetValue(db, abi.GetRegistrationInfoKey(param.SbpName, param.Gid), registerInfo)
	return nil, nil
}

type MethodVote struct {
	MethodName string
}

func (p *MethodVote) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodVote) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodVote) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.VoteQuota, nil
}
func (p *MethodVote) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodVote) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	latestSb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	if block.Amount.Sign() != 0 || (types.IsContractAddr(block.AccountAddress) && !fork.IsStemFork(latestSb.Height)) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamVote)
	if err := abi.ABIGovernance.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameVoteV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	if !checkRegisterAndVoteParam(param.Gid, param.SbpName) {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameVoteV3 {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.SbpName)
	} else {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, param.Gid, param.SbpName)
	}
	return nil
}

func (p *MethodVote) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamVote)
	abi.ABIGovernance.UnpackMethod(param, p.MethodName, sendBlock.Data)
	if p.MethodName == abi.MethodNameVoteV3 {
		param.Gid = types.SNAPSHOT_GID
	}
	consensusGroupInfo, err := abi.GetConsensusGroup(db, param.Gid)
	util.DealWithErr(err)
	if consensusGroupInfo == nil {
		return nil, util.ErrInvalidMethodParam
	}
	active, err := abi.IsActiveRegistration(db, param.SbpName, param.Gid)
	util.DealWithErr(err)
	if !active {
		return nil, util.ErrInvalidMethodParam
	}
	voteKey := abi.GetVoteInfoKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := abi.ABIGovernance.PackVariable(abi.VariableNameVoteInfo, param.SbpName)
	util.SetValue(db, voteKey, voteStatus)
	return nil, nil
}

type MethodCancelVote struct {
	MethodName string
}

func (p *MethodCancelVote) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelVote) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}
func (p *MethodCancelVote) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.CancelVoteQuota, nil
}
func (p *MethodCancelVote) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodCancelVote) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	latestSb, err := db.LatestSnapshotBlock()
	util.DealWithErr(err)
	if block.Amount.Sign() != 0 ||
		(types.IsContractAddr(block.AccountAddress) && !fork.IsStemFork(latestSb.Height)) {
		return util.ErrInvalidMethodParam
	}
	if p.MethodName == abi.MethodNameCancelVoteV3 {
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName)
	} else {
		gid := new(types.Gid)
		err = abi.ABIGovernance.UnpackMethod(gid, p.MethodName, block.Data)
		if err != nil || util.IsDelegateGid(*gid) {
			return util.ErrInvalidMethodParam
		}
		block.Data, _ = abi.ABIGovernance.PackMethod(p.MethodName, *gid)
	}
	return nil
}

func (p *MethodCancelVote) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	gid := new(types.Gid)
	abi.ABIGovernance.UnpackMethod(gid, p.MethodName, sendBlock.Data)
	if p.MethodName == abi.MethodNameCancelVoteV3 {
		gid = &types.SNAPSHOT_GID
	}
	util.SetValue(db, abi.GetVoteInfoKey(sendBlock.AccountAddress, *gid), nil)
	return nil, nil
}
