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
func (p *MethodVote) GetSendQuota(data []byte) (uint64, error) {
	return VoteGas, nil
}
func (p *MethodVote) GetReceiveQuota() uint64 {
	return 0
}

// vote for a super node of a consensus group
func (p *MethodVote) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	param := new(cabi.ParamVote)
	if err := cabi.ABIVote.UnpackMethod(param, cabi.MethodNameVote, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.Gid == types.DELEGATE_GID {
		return errors.New("cannot vote consensus group")
	}

	consensusGroupInfo := cabi.GetConsensusGroup(db, param.Gid)
	if consensusGroupInfo == nil {
		return errors.New("consensus group not exist")
	}

	if !cabi.IsActiveRegistration(db, param.NodeName, param.Gid) {
		return errors.New("registration not exist")
	}

	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.VoteConditionId, cabi.VoteConditionPrefix); !ok {
		return errors.New("consensus group vote condition not exist")
	} else if !condition.checkData(consensusGroupInfo.VoteConditionParam, db, block, param, cabi.MethodNameVote) {
		return errors.New("check vote condition failed")
	}

	block.Data, _ = cabi.ABIVote.PackMethod(cabi.MethodNameVote, param.Gid, param.NodeName)
	return nil
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
func (p *MethodCancelVote) GetSendQuota(data []byte) (uint64, error) {
	return CancelVoteGas, nil
}
func (p *MethodCancelVote) GetReceiveQuota() uint64 {
	return 0
}

// cancel vote for a super node of a consensus group
func (p *MethodCancelVote) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	if block.Amount.Sign() != 0 ||
		!util.IsUserAccount(db, block.AccountAddress) {
		return errors.New("invalid block data")
	}
	gid := new(types.Gid)
	err := cabi.ABIVote.UnpackMethod(gid, cabi.MethodNameCancelVote, block.Data)
	if err != nil || *gid == types.DELEGATE_GID || !IsExistGid(db, *gid) {
		return errors.New("consensus group not exist or cannot cancel vote")
	}
	block.Data, _ = cabi.ABIVote.PackMethod(cabi.MethodNameCancelVote, *gid)
	return nil
}

func (p *MethodCancelVote) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	gid := new(types.Gid)
	cabi.ABIVote.UnpackMethod(gid, cabi.MethodNameCancelVote, sendBlock.Data)
	voteKey := cabi.GetVoteKey(sendBlock.AccountAddress, *gid)
	db.SetStorage(voteKey, nil)
	return nil, nil
}
