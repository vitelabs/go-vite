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
	"strings"
	"time"
)

const (
	jsonRegister = `
	[
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"}]},
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"}]},
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"}]},
		{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"rewardHeight","type":"uint64"},{"name":"cancelHeight","type":"uint64"},{"name":"hisAddrList","type":"address[]"}]},
		{"type":"variable","name":"hisName","inputs":[{"name":"name","type":"string"}]}
	]`

	MethodNameRegister           = "Register"
	MethodNameCancelRegister     = "CancelRegister"
	MethodNameReward             = "Reward"
	MethodNameUpdateRegistration = "UpdateRegistration"
	VariableNameRegistration     = "registration"
	VariableNameHisName          = "hisName"
)

var (
	ABIRegister, _ = abi.JSONToABIContract(strings.NewReader(jsonRegister))
)

type ParamRegister struct {
	Gid      types.Gid
	Name     string
	NodeAddr types.Address
}
type ParamCancelRegister struct {
	Gid  types.Gid
	Name string
}
type ParamReward struct {
	Gid            types.Gid
	Name           string
	BeneficialAddr types.Address
}
type Registration struct {
	Name           string
	NodeAddr       types.Address
	PledgeAddr     types.Address
	Amount         *big.Int
	WithdrawHeight uint64
	RewardHeight   uint64
	CancelHeight   uint64
	HisAddrList    []types.Address
}

func (r *Registration) IsActive() bool {
	return r.CancelHeight == 0
}

func GetRegisterKey(name string, gid types.Gid) []byte {
	var data = make([]byte, types.HashSize)
	copy(data[:types.GidSize], gid[:])
	copy(data[types.GidSize:], types.DataHash([]byte(name)).Bytes()[types.GidSize:])
	return data
}

func GetHisNameKey(addr types.Address, gid types.Gid) []byte {
	var data = make([]byte, types.AddressSize+types.GidSize)
	copy(data[:types.AddressSize], addr[:])
	copy(data[types.AddressSize:], gid[:])
	return data
}

func IsRegisterKey(key []byte) bool {
	if len(key) == types.HashSize {
		return true
	}
	return false
}

func GetRegistration(db StorageDatabase, name string, gid types.Gid) *Registration {
	value := db.GetStorage(&AddressRegister, GetRegisterKey(name, gid))
	registration := new(Registration)
	if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
		if registration.IsActive() {
			return registration
		}
	}
	return nil
}

func GetCandidateList(db StorageDatabase, gid types.Gid) []*Registration {
	defer monitor.LogTime("vm", "GetCandidateList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIterator(&AddressRegister, types.SNAPSHOT_GID.Bytes())
	} else {
		iterator = db.NewStorageIterator(&AddressRegister, gid.Bytes())
	}
	registerList := make([]*Registration, 0)
	if iterator == nil {
		return registerList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if IsRegisterKey(key) {
			registration := new(Registration)
			if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
				if registration.IsActive() {
					registerList = append(registerList, registration)
				}
			}
		}
	}
	return registerList
}

func GetRegistrationList(db StorageDatabase, gid types.Gid, pledgeAddr types.Address) []*Registration {
	defer monitor.LogTime("vm", "GetRegistrationList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIterator(&AddressRegister, types.SNAPSHOT_GID.Bytes())
	} else {
		iterator = db.NewStorageIterator(&AddressRegister, gid.Bytes())
	}
	registerList := make([]*Registration, 0)
	if iterator == nil {
		return registerList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if IsRegisterKey(key) {
			registration := new(Registration)
			if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
				if registration.PledgeAddr == pledgeAddr {
					registerList = append(registerList, registration)
				}
			}
		}
	}
	return registerList
}

type MethodRegister struct {
}

func (p *MethodRegister) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// register to become a super node of a consensus group, lock 1 million ViteToken for 3 month
func (p *MethodRegister) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, RegisterGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(ParamRegister)
	err = ABIRegister.UnpackMethod(param, MethodNameRegister, block.AccountBlock.Data)
	if err != nil || param.Gid == types.DELEGATE_GID {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	if err = checkRegisterData(MethodNameRegister, block, param); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}

func checkRegisterData(methodName string, block *vm_context.VmAccountBlock, param *ParamRegister) error {
	consensusGroupInfo := GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return errors.New("consensus group not exist")
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, RegisterConditionPrefix); !ok {
		return errors.New("register condition id not exist")
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, methodName) {
		return errors.New("register condition not match")
	}
	return nil
}

func (p *MethodRegister) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamRegister)
	ABIRegister.UnpackMethod(param, MethodNameRegister, sendBlock.Data)

	// check old data
	snapshotBlock := block.VmContext.CurrentSnapshotBlock()
	rewardHeight := snapshotBlock.Height
	key := GetRegisterKey(param.Name, param.Gid)
	oldData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)
	var hisAddrList []types.Address
	if len(oldData) > 0 {
		old := new(Registration)
		ABIRegister.UnpackVariable(old, VariableNameRegistration, oldData)
		if old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
			return errors.New("register data exist")
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
		hisAddrList = old.HisAddrList
	}

	// check node addr belong to one name in a consensus group
	hisNameKey := GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	err := ABIRegister.UnpackVariable(hisName, VariableNameHisName, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, hisNameKey))
	if err == nil && *hisName != param.Name {
		// hisName exist
		return errors.New("node address is registered to another name before")
	}
	if err != nil {
		// hisName not exist, update hisName
		hisAddrList = append(hisAddrList, param.NodeAddr)
		hisNameData, _ := ABIRegister.PackVariable(VariableNameHisName, param.Name)
		block.VmContext.SetStorage(hisNameKey, hisNameData)
	}

	registerInfo, _ := ABIRegister.PackVariable(
		VariableNameRegistration,
		param.Name,
		param.NodeAddr,
		sendBlock.AccountAddress,
		sendBlock.Amount,
		getRegisterWithdrawHeight(block.VmContext, param.Gid, snapshotBlock.Height),
		rewardHeight,
		uint64(0),
		hisAddrList)
	block.VmContext.SetStorage(key, registerInfo)
	return nil
}

func getRegisterWithdrawHeight(db vmctxt_interface.VmDatabase, gid types.Gid, currentHeight uint64) uint64 {
	consensusGroupInfo := GetConsensusGroup(db, gid)
	withdrawHeight := getRegisterWithdrawHeightByCondition(consensusGroupInfo.RegisterConditionId, consensusGroupInfo.RegisterConditionParam, currentHeight)
	return withdrawHeight
}

type MethodCancelRegister struct {
}

func (p *MethodCancelRegister) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *MethodCancelRegister) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(ParamCancelRegister)
	err = ABIRegister.UnpackMethod(param, MethodNameCancelRegister, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}

	consensusGroupInfo := GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, errors.New("consensus group not exist")
	}
	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.RegisterConditionId, RegisterConditionPrefix); !ok {
		return quotaLeft, errors.New("consensus group register condition not exist")
	} else if !condition.checkData(consensusGroupInfo.RegisterConditionParam, block, param, MethodNameCancelRegister) {
		return quotaLeft, errors.New("check register condition failed")
	}
	return quotaLeft, nil
}
func (p *MethodCancelRegister) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamCancelRegister)
	ABIRegister.UnpackMethod(param, MethodNameCancelRegister, sendBlock.Data)

	key := GetRegisterKey(param.Name, param.Gid)
	old := new(Registration)
	err := ABIRegister.UnpackVariable(
		old,
		VariableNameRegistration,
		block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
		return errors.New("register not exist or already canceled")
	}

	// update lock amount and loc start height
	snapshotBlock := block.VmContext.CurrentSnapshotBlock()
	registerInfo, _ := ABIRegister.PackVariable(
		VariableNameRegistration,
		param.Name,
		old.NodeAddr,
		old.PledgeAddr,
		helper.Big0,
		uint64(0),
		old.RewardHeight,
		snapshotBlock.Height,
		old.HisAddrList)
	block.VmContext.SetStorage(key, registerInfo)
	// return locked ViteToken
	if old.Amount.Sign() > 0 {
		context.AppendBlock(
			&vm_context.VmAccountBlock{
				util.MakeSendBlock(
					block.AccountBlock,
					sendBlock.AccountAddress,
					ledger.BlockTypeSendCall,
					old.Amount,
					ledger.ViteTokenId,
					context.GetNewBlockHeight(block),
					[]byte{}),
				nil})
	}
	return nil
}

type MethodReward struct {
}

func (p *MethodReward) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// get reward of generating snapshot block
func (p *MethodReward) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, RewardGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() != 0 ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(ParamReward)
	err = ABIRegister.UnpackMethod(param, MethodNameReward, block.AccountBlock.Data)
	if err != nil || !util.IsSnapshotGid(param.Gid) {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	key := GetRegisterKey(param.Name, param.Gid)
	old := new(Registration)
	err = ABIRegister.UnpackVariable(old, VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.ToAddress, key))
	if err != nil || old.PledgeAddr != block.AccountBlock.AccountAddress {
		return quotaLeft, errors.New("registration not exist")
	}
	return quotaLeft, nil
}
func (p *MethodReward) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamReward)
	ABIRegister.UnpackMethod(param, MethodNameReward, sendBlock.Data)
	key := GetRegisterKey(param.Name, param.Gid)
	old := new(Registration)
	err := ABIRegister.UnpackVariable(old, VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || sendBlock.AccountAddress != old.PledgeAddr {
		return errors.New("invalid owner")
	}
	endHeight, reward := CalcReward(block.VmContext, old, false)
	if endHeight != old.RewardHeight {
		registerInfo, _ := ABIRegister.PackVariable(
			VariableNameRegistration,
			old.Name,
			old.NodeAddr,
			old.PledgeAddr,
			old.Amount,
			old.WithdrawHeight,
			endHeight,
			old.CancelHeight,
			old.HisAddrList)
		block.VmContext.SetStorage(key, registerInfo)

		if reward != nil {
			// create reward and return
			context.AppendBlock(
				&vm_context.VmAccountBlock{
					util.MakeSendBlock(
						block.AccountBlock,
						param.BeneficialAddr,
						ledger.BlockTypeSendReward,
						reward,
						ledger.ViteTokenId,
						context.GetNewBlockHeight(block),
						[]byte{}),
					nil})
		}
	}
	return nil
}

func CalcReward(db vmctxt_interface.VmDatabase, old *Registration, total bool) (uint64, *big.Int) {
	if db.CurrentSnapshotBlock().Height < nodeConfig.params.RewardHeightLimit {
		return old.RewardHeight, big.NewInt(0)
	}
	startHeight := old.RewardHeight
	endHeight := db.CurrentSnapshotBlock().Height - nodeConfig.params.RewardHeightLimit
	if !old.IsActive() {
		endHeight = helper.Min(endHeight, old.CancelHeight)
	}
	if !total {
		endHeight = helper.Min(endHeight, startHeight+MaxRewardCount)
	}
	if endHeight <= startHeight {
		return old.RewardHeight, big.NewInt(0)
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
		return endHeight, reward
	}
	return endHeight, big.NewInt(0)
}

type MethodUpdateRegistration struct {
}

func (p *MethodUpdateRegistration) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// update registration info
func (p *MethodUpdateRegistration) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, UpdateRegistrationGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(ParamRegister)
	err = ABIRegister.UnpackMethod(param, MethodNameUpdateRegistration, block.AccountBlock.Data)
	if err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}

	if err = checkRegisterData(MethodNameUpdateRegistration, block, param); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func (p *MethodUpdateRegistration) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamRegister)
	ABIRegister.UnpackMethod(param, MethodNameUpdateRegistration, sendBlock.Data)

	key := GetRegisterKey(param.Name, param.Gid)
	old := new(Registration)
	err := ABIRegister.UnpackVariable(old, VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || !old.IsActive() || old.PledgeAddr != sendBlock.AccountAddress {
		return errors.New("register not exist or already canceled")
	}
	// check node addr belong to one name in a consensus group
	hisNameKey := GetHisNameKey(param.NodeAddr, param.Gid)
	hisName := new(string)
	err = ABIRegister.UnpackVariable(hisName, VariableNameHisName, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, hisNameKey))
	if err == nil && *hisName != param.Name {
		// hisName exist
		return errors.New("node address is registered to another name before")
	}
	if err != nil {
		// hisName not exist, update hisName
		old.HisAddrList = append(old.HisAddrList, param.NodeAddr)
		hisNameData, _ := ABIRegister.PackVariable(VariableNameHisName, param.Name)
		block.VmContext.SetStorage(hisNameKey, hisNameData)
	}
	registerInfo, _ := ABIRegister.PackVariable(
		VariableNameRegistration,
		old.Name,
		param.NodeAddr,
		old.PledgeAddr,
		old.Amount,
		old.WithdrawHeight,
		old.RewardHeight,
		old.CancelHeight,
		old.HisAddrList)
	block.VmContext.SetStorage(key, registerInfo)
	return nil
}
