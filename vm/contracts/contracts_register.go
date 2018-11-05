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
)

type MethodRegister struct {
}

func (p *MethodRegister) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodRegister) GetRefundData() []byte {
	return []byte{1}
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *MethodRegister) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, RegisterGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(cabi.ParamRegister)
	if err = cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameRegister, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	if param.Gid == types.DELEGATE_GID {
		return quotaLeft, errors.New("cannot register consensus group")
	}
	if err = checkRegisterData(cabi.MethodNameRegister, db, block, param); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

func checkRegisterData(methodName string, db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, param *cabi.ParamRegister) error {
	consensusGroupInfo := cabi.GetConsensusGroup(db, param.Gid)
	if consensusGroupInfo == nil {
		return errors.New("consensus group not exist")
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, cabi.RegisterConditionPrefix); !ok {
		return errors.New("register condition id not exist")
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, db, block, param, methodName) {
		return errors.New("register condition not match")
	}
	return nil
}

func (p *MethodRegister) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamRegister)
	cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameRegister, sendBlock.Data)

	// Registration is not exist
	// or registration is not active and belongs to sender account
	snapshotBlock := db.CurrentSnapshotBlock()
	rewardHeight := snapshotBlock.Height
	key := cabi.GetRegisterKey(param.Name, param.Gid)
	oldData := db.GetStorage(&block.AccountAddress, key)
	var hisAddrList []types.Address
	if len(oldData) > 0 {
		old := new(types.Registration)
		cabi.ABIRegister.UnpackVariable(old, cabi.VariableNameRegistration, oldData)
		if old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
			return nil, errors.New("register data exist")
		}
		// reward of last being a super node is not drained
		if old.RewardHeight < old.CancelHeight {
			rewardHeight = old.RewardHeight
		}
		hisAddrList = old.HisAddrList
	}

	// Node addr belong to one name in a consensus group
	hisNameKey := cabi.GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	err := cabi.ABIRegister.UnpackVariable(hisName, cabi.VariableNameHisName, db.GetStorage(&block.AccountAddress, hisNameKey))
	if err == nil && *hisName != param.Name {
		return nil, errors.New("node address is registered to another name before")
	}
	if err != nil {
		// hisName not exist, update hisName
		hisAddrList = append(hisAddrList, param.NodeAddr)
		hisNameData, _ := cabi.ABIRegister.PackVariable(cabi.VariableNameHisName, param.Name)
		db.SetStorage(hisNameKey, hisNameData)
	}

	registerInfo, _ := cabi.ABIRegister.PackVariable(
		cabi.VariableNameRegistration,
		param.Name,
		param.NodeAddr,
		sendBlock.AccountAddress,
		sendBlock.Amount,
		getRegisterWithdrawHeight(db, param.Gid, snapshotBlock.Height),
		rewardHeight,
		uint64(0),
		hisAddrList)
	db.SetStorage(key, registerInfo)
	return nil, nil
}

func getRegisterWithdrawHeight(db vmctxt_interface.VmDatabase, gid types.Gid, currentHeight uint64) uint64 {
	consensusGroupInfo := cabi.GetConsensusGroup(db, gid)
	withdrawHeight := getRegisterWithdrawHeightByCondition(consensusGroupInfo.RegisterConditionId, consensusGroupInfo.RegisterConditionParam, currentHeight)
	return withdrawHeight
}

type MethodCancelRegister struct {
}

func (p *MethodCancelRegister) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (p *MethodCancelRegister) GetRefundData() []byte {
	return []byte{2}
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *MethodCancelRegister) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(cabi.ParamCancelRegister)
	if err = cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameCancelRegister, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}

	consensusGroupInfo := cabi.GetConsensusGroup(db, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, errors.New("consensus group not exist")
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, cabi.RegisterConditionPrefix); !ok {
		return quotaLeft, errors.New("consensus group register condition not exist")
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, db, block, param, cabi.MethodNameCancelRegister) {
		return quotaLeft, errors.New("check register condition failed")
	}
	return quotaLeft, nil
}
func (p *MethodCancelRegister) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamCancelRegister)
	cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameCancelRegister, sendBlock.Data)

	snapshotBlock := db.CurrentSnapshotBlock()

	key := cabi.GetRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err := cabi.ABIRegister.UnpackVariable(
		old,
		cabi.VariableNameRegistration,
		db.GetStorage(&block.AccountAddress, key))
	if err != nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress || old.WithdrawHeight > snapshotBlock.Height {
		return nil, errors.New("registration status error")
	}

	// update lock amount and loc start height
	registerInfo, _ := cabi.ABIRegister.PackVariable(
		cabi.VariableNameRegistration,
		old.Name,
		old.NodeAddr,
		old.PledgeAddr,
		helper.Big0,
		uint64(0),
		old.RewardHeight,
		snapshotBlock.Height,
		old.HisAddrList)
	db.SetStorage(key, registerInfo)
	// return locked ViteToken
	if old.Amount.Sign() > 0 {
		return []*SendBlock{
			{
				block,
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

func (p *MethodReward) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodReward) GetRefundData() []byte {
	return []byte{3}
}

// get reward of generating snapshot block
func (p *MethodReward) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, RewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() != 0 ||
		!IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(cabi.ParamReward)
	if err = cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameReward, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	if !util.IsSnapshotGid(param.Gid) {
		return quotaLeft, errors.New("consensus group has no reward")
	}
	return quotaLeft, nil
}
func (p *MethodReward) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamReward)
	cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameReward, sendBlock.Data)
	key := cabi.GetRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err := cabi.ABIRegister.UnpackVariable(old, cabi.VariableNameRegistration, db.GetStorage(&block.AccountAddress, key))
	if err != nil || sendBlock.AccountAddress != old.PledgeAddr {
		return nil, errors.New("invalid owner")
	}
	_, endHeight, reward := CalcReward(db, old, false)
	if endHeight != old.RewardHeight {
		registerInfo, _ := cabi.ABIRegister.PackVariable(
			cabi.VariableNameRegistration,
			old.Name,
			old.NodeAddr,
			old.PledgeAddr,
			old.Amount,
			old.WithdrawHeight,
			endHeight,
			old.CancelHeight,
			old.HisAddrList)
		db.SetStorage(key, registerInfo)

		if reward != nil && reward.Sign() > 0 {
			return []*SendBlock{
				{
					block,
					param.BeneficialAddr,
					ledger.BlockTypeSendReward,
					reward,
					ledger.ViteTokenId,
					[]byte{},
				},
			}, nil
		}
	}
	return nil, nil
}

func CalcReward(db vmctxt_interface.VmDatabase, old *types.Registration, total bool) (uint64, uint64, *big.Int) {
	if db.CurrentSnapshotBlock().Height < nodeConfig.params.RewardHeightLimit {
		return old.RewardHeight, old.RewardHeight, big.NewInt(0)
	}
	// startHeight exclusive, endHeight inclusive
	startHeight := old.RewardHeight
	endHeight := db.CurrentSnapshotBlock().Height - nodeConfig.params.RewardHeightLimit
	if !old.IsActive() {
		endHeight = helper.Min(endHeight, old.CancelHeight)
	}
	if !total {
		endHeight = helper.Min(endHeight, startHeight+MaxRewardCount)
	}
	if endHeight <= startHeight {
		return old.RewardHeight, old.RewardHeight, big.NewInt(0)
	}
	count := endHeight - startHeight

	startHeight = startHeight + 1
	rewardCount := uint64(0)
	producerMap := make(map[types.Address]interface{})
	for _, producer := range old.HisAddrList {
		producerMap[producer] = struct{}{}
	}
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
			if _, ok := producerMap[block.Producer()]; ok {
				rewardCount++
			}
		}
	}
	if rewardCount > 0 {
		reward := new(big.Int).SetUint64(rewardCount)
		reward.Mul(rewardPerBlock, reward)
		return old.RewardHeight, endHeight, reward
	}
	return old.RewardHeight, endHeight, big.NewInt(0)
}

type MethodUpdateRegistration struct {
}

func (p *MethodUpdateRegistration) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodUpdateRegistration) GetRefundData() []byte {
	return []byte{4}
}

// update registration info
func (p *MethodUpdateRegistration) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, UpdateRegistrationGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(cabi.ParamRegister)
	if err = cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameUpdateRegistration, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}

	if err = checkRegisterData(cabi.MethodNameUpdateRegistration, db, block, param); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func (p *MethodUpdateRegistration) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamRegister)
	cabi.ABIRegister.UnpackMethod(param, cabi.MethodNameUpdateRegistration, sendBlock.Data)

	key := cabi.GetRegisterKey(param.Name, param.Gid)
	old := new(types.Registration)
	err := cabi.ABIRegister.UnpackVariable(old, cabi.VariableNameRegistration, db.GetStorage(&block.AccountAddress, key))
	if err != nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
		return nil, errors.New("register not exist or already canceled")
	}
	// check node addr belong to one name in a consensus group
	hisNameKey := cabi.GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	err = cabi.ABIRegister.UnpackVariable(hisName, cabi.VariableNameHisName, db.GetStorage(&block.AccountAddress, hisNameKey))
	if err == nil && *hisName != param.Name {
		// hisName exist
		return nil, errors.New("node address is registered to another name before")
	}
	if err != nil {
		// hisName not exist, update hisName
		old.HisAddrList = append(old.HisAddrList, param.NodeAddr)
		hisNameData, _ := cabi.ABIRegister.PackVariable(cabi.VariableNameHisName, param.Name)
		db.SetStorage(hisNameKey, hisNameData)
	}
	registerInfo, _ := cabi.ABIRegister.PackVariable(
		cabi.VariableNameRegistration,
		old.Name,
		param.NodeAddr,
		old.PledgeAddr,
		old.Amount,
		old.WithdrawHeight,
		old.RewardHeight,
		old.CancelHeight,
		old.HisAddrList)
	db.SetStorage(key, registerInfo)
	return nil, nil
}
