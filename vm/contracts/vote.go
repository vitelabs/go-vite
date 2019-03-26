package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type MethodVote struct {
}

func (p *MethodVote) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodVote) GetRefundData() []byte {
	return []byte{5}
}
func (p *MethodVote) GetSendQuota(data []byte) (uint64, error) {
	return VoteGas, nil
}

// vote for a super node of a consensus group
func (p *MethodVote) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 || !util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamVote)
	if err := abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameVote, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if !checkRegisterAndVoteParam(param.Gid, param.NodeName) {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, param.Gid, param.NodeName)
	return nil
}

func (p *MethodVote) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	param := new(abi.ParamVote)
	abi.ABIConsensusGroup.UnpackMethod(param, abi.MethodNameVote, sendBlock.Data)
	consensusGroupInfo, err := abi.GetConsensusGroup(db, param.Gid)
	util.DealWithErr(err)
	if consensusGroupInfo == nil {
		return nil, util.ErrInvalidMethodParam
	}
	if active, err := abi.IsActiveRegistration(db, param.NodeName, param.Gid); err != nil || !active {
		return nil, util.ErrInvalidMethodParam
	}
	voteKey := abi.GetVoteKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameVoteStatus, param.NodeName)
	db.SetValue(voteKey, voteStatus)
	return nil, nil
}

type MethodCancelVote struct {
}

func (p *MethodCancelVote) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelVote) GetRefundData() []byte {
	return []byte{6}
}
func (p *MethodCancelVote) GetSendQuota(data []byte) (uint64, error) {
	return CancelVoteGas, nil
}

// cancel vote for a super node of a consensus group
func (p *MethodCancelVote) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	gid := new(types.Gid)
	err := abi.ABIConsensusGroup.UnpackMethod(gid, abi.MethodNameCancelVote, block.Data)
	if err != nil || *gid == types.DELEGATE_GID {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelVote, *gid)
	return nil
}

func (p *MethodCancelVote) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	gid := new(types.Gid)
	abi.ABIConsensusGroup.UnpackMethod(gid, abi.MethodNameCancelVote, sendBlock.Data)
	db.SetValue(abi.GetVoteKey(sendBlock.AccountAddress, *gid), nil)
	return nil, nil
}
