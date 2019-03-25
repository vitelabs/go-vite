package contracts

/*type MethodCreateConsensusGroup struct{}

func (p *MethodCreateConsensusGroup) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCreateConsensusGroup) GetRefundData() []byte {
	return []byte{1}
}

func (p *MethodCreateConsensusGroup) GetQuota() uint64 {
	return CreateConsensusGroupGas
}

func (p *MethodCreateConsensusGroup) DoSend(db vm_db.VmDb, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!util.IsViteToken(block.TokenId) ||
		!util.IsUserAccount(db) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(types.ConsensusGroupInfo)
	err = abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameCreateConsensusGroup, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	if err := CheckCreateConsensusGroupData(db, param); err != nil {
		return quotaLeft, err
	}
	gid := abi.NewGid(block.AccountAddress, block.Height, block.PrevHash, block.SnapshotHash)
	if IsExistGid(db, gid) {
		return quotaLeft, errors.New("consensus group id already exists")
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(
		abi.MethodNameCreateConsensusGroup,
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
	return quotaLeft, nil
}
func CheckCreateConsensusGroupData(db vm_db.VmDb, param *types.ConsensusGroupInfo) error {
	if param.NodeCount < cgNodeCountMin || param.NodeCount > cgNodeCountMax ||
		param.Interval < cgIntervalMin || param.Interval > cgIntervalMax ||
		param.PerCount < cgPerCountMin || param.PerCount > cgPerCountMax ||
		// no overflow
		param.PerCount*param.Interval < cgPerIntervalMin || param.PerCount*param.Interval > cgPerIntervalMax ||
		param.RandCount > param.NodeCount ||
		(param.RandCount > 0 && param.RandRank < param.NodeCount) {
		return errors.New("invalid consensus group param")
	}
	if abi.GetTokenById(db, param.CountingTokenId) == nil {
		return errors.New("counting token id not exist")
	}
	if err := checkCondition(db, param.RegisterConditionId, param.RegisterConditionParam, abi.RegisterConditionPrefix); err != nil {
		return err
	}
	if err := checkCondition(db, param.VoteConditionId, param.VoteConditionParam, abi.VoteConditionPrefix); err != nil {
		return err
	}
	return nil
}
func checkCondition(db vm_db.VmDb, conditionId uint8, conditionParam []byte, conditionIdPrefix abi.ConditionCode) error {
	condition, ok := getConsensusGroupCondition(conditionId, conditionIdPrefix)
	if !ok {
		return errors.New("condition id not exist")
	}
	if ok := condition.checkParam(conditionParam, db); !ok {
		return errors.New("invalid condition param")
	}
	return nil
}
func (p *MethodCreateConsensusGroup) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	param := new(types.ConsensusGroupInfo)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameCreateConsensusGroup, sendBlock.Data)
	key := abi.GetConsensusGroupKey(param.Gid)
	if len(db.GetValue(key)) > 0 {
		return nil, util.ErrIdCollision
	}
	groupInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameConsensusGroupInfo,
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
		globalStatus.SnapshotBlock.Height+nodeConfig.params.CreateConsensusGroupPledgeHeight)
	db.SetValue(key, groupInfo)
	return nil, nil
}

type MethodCancelConsensusGroup struct{}

func (p *MethodCancelConsensusGroup) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelConsensusGroup) GetRefundData() []byte {
	return []byte{2}
}

func (p *MethodCancelConsensusGroup) GetQuota() uint64 {
	return CancelConsensusGroupGas
}

// Cancel consensus group and get pledge back.
// A canceled consensus group(no-active) will not generate contract blocks after cancel receive block is confirmed.
// Consensus group name is kept even if canceled.
func (p *MethodCancelConsensusGroup) DoSend(db vm_db.VmDb, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	err = abi.ABIConsensusGroup.UnpackMethod(gid, abi.MethodNameCancelConsensusGroup, block.Data)
	if err != nil {
		return quotaLeft, err
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelConsensusGroup, *gid)
	return quotaLeft, nil
}
func (p *MethodCancelConsensusGroup) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	gid := new(types.Gid)
	abi.ABIConsensusGroup.UnpackMethod(gid, abi.MethodNameCancelConsensusGroup, sendBlock.Data)
	key := abi.GetConsensusGroupKey(*gid)
	groupInfo := abi.GetConsensusGroup(db, *gid)
	if groupInfo == nil ||
		!groupInfo.IsActive() ||
		groupInfo.WithdrawHeight > globalStatus.SnapshotBlock.Height {
		return nil, errors.New("pledge not yet due")
	}
	newGroupInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameConsensusGroupInfo,
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
	db.SetValue(key, newGroupInfo)
	if groupInfo.PledgeAmount.Sign() > 0 {
		return []*SendBlock{
			{
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

func (p *MethodReCreateConsensusGroup) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodReCreateConsensusGroup) GetRefundData() []byte {
	return []byte{3}
}

func (p *MethodReCreateConsensusGroup) GetQuota() uint64 {
	return ReCreateConsensusGroupGas
}

// Pledge again for a canceled consensus group.
// A consensus group will start generate contract blocks after recreate receive block is confirmed.
func (p *MethodReCreateConsensusGroup) DoSend(db vm_db.VmDb, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(createConsensusGroupPledgeAmount) != 0 ||
		!util.IsViteToken(block.TokenId) ||
		!util.IsUserAccount(db) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	if err = abi.ABIConsensusGroup.UnpackMethod(gid, abi.MethodNameReCreateConsensusGroup, block.Data); err != nil {
		return quotaLeft, err
	}
	if groupInfo := abi.GetConsensusGroup(db, *gid); groupInfo == nil ||
		block.AccountAddress != groupInfo.Owner ||
		groupInfo.IsActive() {
		return quotaLeft, errors.New("invalid group info or owner or status")
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameReCreateConsensusGroup, *gid)
	return quotaLeft, nil
}
func (p *MethodReCreateConsensusGroup) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	gid := new(types.Gid)
	abi.ABIConsensusGroup.UnpackMethod(gid, abi.MethodNameReCreateConsensusGroup, sendBlock.Data)
	key := abi.GetConsensusGroupKey(*gid)
	groupInfo := abi.GetConsensusGroup(db, *gid)
	if groupInfo == nil ||
		groupInfo.IsActive() {
		return nil, errors.New("consensus group is active")
	}
	newGroupInfo, _ := abi.ABIConsensusGroup.PackVariable(
		abi.VariableNameConsensusGroupInfo,
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
		globalStatus.SnapshotBlock.Height+nodeConfig.params.CreateConsensusGroupPledgeHeight)
	db.SetValue(key, newGroupInfo)
	return nil, nil
}

type createConsensusGroupCondition interface {
	checkParam(param []byte, db vm_db.VmDb) bool
}

var SimpleCountingRuleList = map[abi.ConditionCode]createConsensusGroupCondition{
	abi.RegisterConditionOfPledge: &registerConditionOfPledge{},
	abi.VoteConditionOfDefault:    &voteConditionOfDefault{},
}

func getConsensusGroupCondition(conditionId uint8, conditionIdPrefix abi.ConditionCode) (createConsensusGroupCondition, bool) {
	condition, ok := SimpleCountingRuleList[conditionIdPrefix+abi.ConditionCode(conditionId)]
	return condition, ok
}

type registerConditionOfPledge struct{}

func (c registerConditionOfPledge) checkParam(param []byte, db vm_db.VmDb) bool {
	v := new(abi.VariableConditionRegisterOfPledge)
	err := abi.ABIConsensusGroup.UnpackVariable(v, abi.VariableNameConditionRegisterOfPledge, param)
	if err != nil ||
		abi.GetTokenById(db, v.PledgeToken) == nil ||
		v.PledgeAmount.Sign() == 0 ||
		v.PledgeHeight < nodeConfig.params.RegisterMinPledgeHeight {
		return false
	}
	return true
}

type voteConditionOfDefault struct{}

func (c voteConditionOfDefault) checkParam(param []byte, db vm_db.VmDb) bool {
	if len(param) != 0 {
		return false
	}
	return true
}*/
