package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

type MethodVote struct {
}

func (p *MethodVote) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodVote) GetRefundData() []byte {
	return []byte{1}
}
func (p *MethodVote) GetQuota() uint64 {
	return VoteGas
}

// vote for a super node of a consensus group
func (p *MethodVote) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}

	param := new(cabi.ParamVote)
	if err = cabi.ABIVote.UnpackMethod(param, cabi.MethodNameVote, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	if param.Gid == types.DELEGATE_GID {
		return quotaLeft, errors.New("cannot vote consensus group")
	}

	consensusGroupInfo := cabi.GetConsensusGroup(db, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, errors.New("consensus group not exist")
	}

	if !cabi.IsActiveRegistration(db, param.NodeName, param.Gid) {
		return quotaLeft, errors.New("registration not exist")
	}

	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.VoteConditionId, cabi.VoteConditionPrefix); !ok {
		return quotaLeft, errors.New("consensus group vote condition not exist")
	} else if !condition.checkData(consensusGroupInfo.VoteConditionParam, db, block, param, cabi.MethodNameVote) {
		return quotaLeft, errors.New("check vote condition failed")
	}

	return quotaLeft, nil
}

func (p *MethodVote) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamVote)
	cabi.ABIVote.UnpackMethod(param, cabi.MethodNameVote, sendBlock.Data)
	voteKey := cabi.GetVoteKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := cabi.ABIVote.PackVariable(cabi.VariableNameVoteStatus, param.NodeName)
	db.SetStorage(voteKey, voteStatus)
	return nil, nil
}

type MethodCancelVote struct {
}

func (p *MethodCancelVote) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelVote) GetRefundData() []byte {
	return []byte{2}
}
func (p *MethodCancelVote) GetQuota() uint64 {
	return CancelVoteGas
}

// cancel vote for a super node of a consensus group
func (p *MethodCancelVote) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, p.GetQuota())
	if err != nil {
		return quotaLeft, err
	}

	if block.Amount.Sign() != 0 ||
		!IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	err = cabi.ABIVote.UnpackMethod(gid, cabi.MethodNameCancelVote, block.Data)
	if err != nil || *gid == types.DELEGATE_GID || !IsExistGid(db, *gid) {
		return quotaLeft, errors.New("consensus group not exist or cannot cancel vote")
	}
	return quotaLeft, nil
}

func (p *MethodCancelVote) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	gid := new(types.Gid)
	cabi.ABIVote.UnpackMethod(gid, cabi.MethodNameCancelVote, sendBlock.Data)
	voteKey := cabi.GetVoteKey(sendBlock.AccountAddress, *gid)
	db.SetStorage(voteKey, nil)
	return nil, nil
}
