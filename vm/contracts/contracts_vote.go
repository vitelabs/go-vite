package contracts

import (
	"errors"
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
	jsonVote = `
	[
		{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeName","type":"string"}]},
		{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"voteStatus","inputs":[{"name":"nodeName","type":"string"}]}
	]`

	MethodNameVote         = "Vote"
	MethodNameCancelVote   = "CancelVote"
	VariableNameVoteStatus = "voteStatus"
)

var (
	ABIVote, _ = abi.JSONToABIContract(strings.NewReader(jsonVote))
)

type ParamVote struct {
	Gid      types.Gid
	NodeName string
}

func GetVoteKey(addr types.Address, gid types.Gid) []byte {
	return append(gid.Bytes(), addr.Bytes()...)
}

func GetAddrFromVoteKey(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[types.GidSize:])
	return addr
}

func GetVote(db StorageDatabase, gid types.Gid, addr types.Address) *types.VoteInfo {
	defer monitor.LogTime("vm", "GetVote", time.Now())
	data := db.GetStorage(&AddressVote, GetVoteKey(addr, gid))
	if len(data) > 0 {
		nodeName := new(string)
		ABIVote.UnpackVariable(nodeName, VariableNameVoteStatus, data)
		return &types.VoteInfo{addr, *nodeName}
	}
	return nil
}

func GetVoteList(db StorageDatabase, gid types.Gid) []*types.VoteInfo {
	defer monitor.LogTime("vm", "GetVoteList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIterator(&AddressVote, types.SNAPSHOT_GID.Bytes())
	} else {
		iterator = db.NewStorageIterator(&AddressVote, gid.Bytes())
	}
	voteInfoList := make([]*types.VoteInfo, 0)
	if iterator == nil {
		return voteInfoList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		voterAddr := GetAddrFromVoteKey(key)
		nodeName := new(string)
		if err := ABIVote.UnpackVariable(nodeName, VariableNameVoteStatus, value); err == nil {
			voteInfoList = append(voteInfoList, &types.VoteInfo{voterAddr, *nodeName})
		}
	}
	return voteInfoList
}

type MethodVote struct {
}

func (p *MethodVote) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodVote) GetRefundData() []byte {
	return []byte{1}
}

// vote for a super node of a consensus group
func (p *MethodVote) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, VoteGas)
	if err != nil {
		return quotaLeft, err
	}

	param := new(ParamVote)
	if err = ABIVote.UnpackMethod(param, MethodNameVote, block.AccountBlock.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	if param.Gid == types.DELEGATE_GID {
		return quotaLeft, errors.New("cannot vote consensus group")
	}

	consensusGroupInfo := GetConsensusGroup(block.VmContext, param.Gid)
	if consensusGroupInfo == nil {
		return quotaLeft, errors.New("consensus group not exist")
	}

	if !IsActiveRegistration(block.VmContext, param.NodeName, param.Gid) {
		return quotaLeft, errors.New("registration not exist")
	}

	if condition, ok := getConsensusGroupCondition(consensusGroupInfo.VoteConditionId, VoteConditionPrefix); !ok {
		return quotaLeft, errors.New("consensus group vote condition not exist")
	} else if !condition.checkData(consensusGroupInfo.VoteConditionParam, block, param, MethodNameVote) {
		return quotaLeft, errors.New("check vote condition failed")
	}

	return quotaLeft, nil
}

func (p *MethodVote) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamVote)
	ABIVote.UnpackMethod(param, MethodNameVote, sendBlock.Data)
	voteKey := GetVoteKey(sendBlock.AccountAddress, param.Gid)
	voteStatus, _ := ABIVote.PackVariable(VariableNameVoteStatus, param.NodeName)
	block.VmContext.SetStorage(voteKey, voteStatus)
	return nil
}

type MethodCancelVote struct {
}

func (p *MethodCancelVote) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelVote) GetRefundData() []byte {
	return []byte{2}
}

// cancel vote for a super node of a consensus group
func (p *MethodCancelVote) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelVoteGas)
	if err != nil {
		return quotaLeft, err
	}

	if block.AccountBlock.Amount.Sign() != 0 ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	gid := new(types.Gid)
	err = ABIVote.UnpackMethod(gid, MethodNameCancelVote, block.AccountBlock.Data)
	if err != nil || *gid == types.DELEGATE_GID || !IsExistGid(block.VmContext, *gid) {
		return quotaLeft, errors.New("consensus group not exist or cannot cancel vote")
	}
	return quotaLeft, nil
}

func (p *MethodCancelVote) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	gid := new(types.Gid)
	ABIVote.UnpackMethod(gid, MethodNameCancelVote, sendBlock.Data)
	voteKey := GetVoteKey(sendBlock.AccountAddress, *gid)
	block.VmContext.SetStorage(voteKey, nil)
	return nil
}
