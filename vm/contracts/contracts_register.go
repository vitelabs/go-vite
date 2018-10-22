package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
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
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"publicKey","type":"bytes"},{"name":"signature","type":"bytes"}]},
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"publicKey","type":"bytes"},{"name":"signature","type":"bytes"}]},
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"},{"name":"endHeight","type":"uint64"},{"name":"startHeight","type":"uint64"},{"name":"amount","type":"uint256"}]},
		{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"pledgeHeight","type":"uint64"},{"name":"rewardHeight","type":"uint64"},{"name":"cancelHeight","type":"uint64"}]}
	]`

	MethodNameRegister           = "Register"
	MethodNameCancelRegister     = "CancelRegister"
	MethodNameReward             = "Reward"
	MethodNameUpdateRegistration = "UpdateRegistration"
	VariableNameRegistration     = "registration"
)

var (
	ABIRegister, _ = abi.JSONToABIContract(strings.NewReader(jsonRegister))
)

type ParamRegister struct {
	Gid       types.Gid
	Name      string
	NodeAddr  types.Address
	PublicKey []byte
	Signature []byte
}
type ParamCancelRegister struct {
	Gid  types.Gid
	Name string
}
type ParamReward struct {
	Gid            types.Gid
	Name           string
	BeneficialAddr types.Address
	EndHeight      uint64
	StartHeight    uint64
	Amount         *big.Int
}
type Registration struct {
	Name         string
	NodeAddr     types.Address
	PledgeAddr   types.Address
	Amount       *big.Int
	PledgeHeight uint64
	RewardHeight uint64
	CancelHeight uint64
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

func GetRegisterList(db StorageDatabase, gid types.Gid) []*Registration {
	defer monitor.LogTime("vm", "GetRegisterList", time.Now())
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
		_, value, ok := iterator.Next()
		if !ok {
			break
		}
		registration := new(Registration)
		if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
			if registration.IsActive() {
				registerList = append(registerList, registration)
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
	quotaLeft, err := util.UseQuota(quotaLeft, registerGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
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

	if types.PubkeyToAddress(param.PublicKey) != param.NodeAddr {
		return errors.New("invalid public key")
	}

	if verified, err := crypto.VerifySig(
		param.PublicKey,
		GetRegisterMessageForSignature(block.AccountBlock.AccountAddress, param.Gid),
		param.Signature); !verified {
		return err
	}
	return nil
}

func (p *MethodRegister) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamRegister)
	ABIRegister.UnpackMethod(param, MethodNameRegister, sendBlock.Data)
	// two registration in one consensus group do not share node address
	registrationList := GetRegisterList(block.VmContext, param.Gid)
	for _, registration := range registrationList {
		if registration.NodeAddr == param.NodeAddr {
			return errors.New("duplicate node address")
		}
	}
	snapshotBlock := block.VmContext.CurrentSnapshotBlock()
	rewardHeight := snapshotBlock.Height
	key := GetRegisterKey(param.Name, param.Gid)
	oldData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key)
	if len(oldData) > 0 {
		old := new(Registration)
		ABIRegister.UnpackVariable(old, VariableNameRegistration, oldData)
		if old.IsActive() {
			// duplicate register
			return errors.New("duplicate register")
		}
		// reward of last being a super node is not drained
		rewardHeight = old.RewardHeight
	}
	registerInfo, _ := ABIRegister.PackVariable(
		VariableNameRegistration,
		param.Name,
		param.NodeAddr,
		sendBlock.AccountAddress,
		sendBlock.Amount,
		snapshotBlock.Height,
		rewardHeight,
		uint64(0))
	block.VmContext.SetStorage(key, registerInfo)
	return nil
}

type MethodCancelRegister struct {
}

func (p *MethodCancelRegister) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel register to become a super node of a consensus group after registered for 3 month, get 100w ViteToken back
func (p *MethodCancelRegister) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, cancelRegisterGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
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
	if err != nil || !old.IsActive() {
		return errors.New("register not exist or already canceled")
	}

	// update lock amount and loc start height
	snapshotBlock := block.VmContext.CurrentSnapshotBlock()
	registerInfo, _ := ABIRegister.PackVariable(
		VariableNameRegistration,
		param.Name, old.NodeAddr,
		old.PledgeAddr,
		helper.Big0,
		uint64(0),
		old.RewardHeight,
		snapshotBlock.Height)
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
	quotaLeft, err := util.UseQuota(quotaLeft, rewardGas)
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
	if err != nil || block.AccountBlock.AccountAddress != old.PledgeAddr {
		return quotaLeft, errors.New("invalid register owner")
	}
	if block.VmContext.CurrentSnapshotBlock().Height < rewardHeightLimit {
		return quotaLeft, errors.New("reward height limit not reached")
	}

	if param.EndHeight == 0 {
		param.EndHeight = block.VmContext.CurrentSnapshotBlock().Height - rewardHeightLimit
		if !old.IsActive() {
			param.EndHeight = helper.Min(param.EndHeight, old.CancelHeight)
		}
	} else if param.EndHeight > block.VmContext.CurrentSnapshotBlock().Height-rewardHeightLimit ||
		(!old.IsActive() && param.EndHeight > old.CancelHeight) {
		return quotaLeft, errors.New("invalid end height")
	}

	if param.StartHeight == 0 {
		param.StartHeight = old.RewardHeight
	}

	if param.EndHeight <= param.StartHeight {
		return quotaLeft, errors.New("invalid end height")
	}

	count := param.EndHeight - param.StartHeight
	// avoid uint64 overflow
	if count > maxRewardCount {
		return quotaLeft, errors.New("height gap overflow")
	}
	if quotaLeft, err = util.UseQuota(quotaLeft, ((count+dbPageSize-1)/dbPageSize)*calcRewardGasPerPage); err != nil {
		return quotaLeft, err
	}

	// calc snapshot block produce reward between param.StartHeight(excluded) and param.EndHeight(included)
	calcReward(block.VmContext, old.NodeAddr, param.StartHeight, count, param.Amount)
	block.AccountBlock.Data, err = ABIRegister.PackMethod(
		MethodNameReward,
		param.Gid,
		param.Name,
		param.BeneficialAddr,
		param.EndHeight,
		param.StartHeight,
		param.Amount)
	if err != nil {
		return quotaLeft, err
	}
	if quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft); err != nil {
		return quotaLeft, err
	}
	return quotaLeft, nil
}
func calcReward(db vmctxt_interface.VmDatabase, producer types.Address, startHeight uint64, count uint64, reward *big.Int) {
	startHeight = startHeight + 1
	var rewardCount uint64
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
			if block.Producer() == producer {
				rewardCount++
			}
		}
	}
	reward.SetUint64(rewardCount)
	reward.Mul(rewardPerBlock, reward)
}
func (p *MethodReward) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamReward)
	ABIRegister.UnpackMethod(param, MethodNameReward, sendBlock.Data)
	key := GetRegisterKey(param.Name, param.Gid)
	old := new(Registration)
	err := ABIRegister.UnpackVariable(old, VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || old.RewardHeight > param.StartHeight || sendBlock.AccountAddress != old.PledgeAddr {
		return errors.New("invalid owner or start height")
	}
	if !old.IsActive() {
		if param.EndHeight > old.CancelHeight {
			return errors.New("invalid end height, supposed to be lower than cancel height")
		} else {
			// get reward partly, update storage
			registerInfo, _ := ABIRegister.PackVariable(
				VariableNameRegistration,
				old.Name,
				old.NodeAddr,
				old.PledgeAddr,
				old.Amount,
				old.PledgeHeight,
				param.EndHeight,
				old.CancelHeight)
			block.VmContext.SetStorage(key, registerInfo)
		}
	} else {
		registerInfo, _ := ABIRegister.PackVariable(
			VariableNameRegistration,
			old.Name,
			old.NodeAddr,
			old.PledgeAddr,
			old.Amount,
			old.PledgeHeight,
			param.EndHeight,
			old.CancelHeight)
		block.VmContext.SetStorage(key, registerInfo)
	}

	if param.Amount.Sign() > 0 {
		// create reward and return
		context.AppendBlock(
			&vm_context.VmAccountBlock{
				util.MakeSendBlock(
					block.AccountBlock,
					param.BeneficialAddr,
					ledger.BlockTypeSendReward,
					param.Amount,
					ledger.ViteTokenId,
					context.GetNewBlockHeight(block),
					[]byte{}),
				nil})
	}
	return nil
}

type MethodUpdateRegistration struct {
}

func (p *MethodUpdateRegistration) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// update registration info
func (p *MethodUpdateRegistration) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, updateRegistrationGas)
	if err != nil {
		return quotaLeft, err
	}
	quotaLeft, err = util.UseQuotaForData(block.AccountBlock.Data, quotaLeft)
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
	// two registration in one consensus group do not share node address
	registrationList := GetRegisterList(block.VmContext, param.Gid)
	for _, registration := range registrationList {
		if registration.NodeAddr == param.NodeAddr {
			return errors.New("duplicate node address")
		}
	}
	key := GetRegisterKey(param.Name, param.Gid)
	old := new(Registration)
	err := ABIRegister.UnpackVariable(old, VariableNameRegistration, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, key))
	if err != nil || !old.IsActive() {
		return errors.New("register not exist or already canceled")
	}
	registerInfo, _ := ABIRegister.PackVariable(
		VariableNameRegistration,
		old.Name, param.NodeAddr,
		old.PledgeAddr,
		old.Amount,
		old.PledgeHeight,
		old.RewardHeight,
		old.CancelHeight)
	block.VmContext.SetStorage(key, registerInfo)
	return nil
}
