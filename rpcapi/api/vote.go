package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_context"
)

type VoteApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewVoteApi(vite *vite.Vite) *VoteApi {
	return &VoteApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/vote_api"),
	}
}

func (v VoteApi) String() string {
	return "VoteApi"
}

func (v *VoteApi) GetVoteData(gid types.Gid, name string) ([]byte, error) {
	return abi.ABIVote.PackMethod(abi.MethodNameVote, gid, name)
}
func (v *VoteApi) GetCancelVoteData(gid types.Gid) ([]byte, error) {
	return abi.ABIVote.PackMethod(abi.MethodNameCancelVote, gid)
}

var (
	NodeStatusActive   uint8 = 1
	NodeStatusInActive uint8 = 2
)

type VoteInfo struct {
	Name       string `json:"nodeName"`
	NodeStatus uint8  `json:"nodeStatus"`
	Balance    string `json:"balance"`
}

func (v *VoteApi) GetVoteInfo(gid types.Gid, addr types.Address) (*VoteInfo, error) {
	vmContext, err := vm_context.NewVmContext(v.chain, nil, nil, &addr)
	if err != nil {
		return nil, err
	}
	if voteInfo := abi.GetVote(vmContext, gid, addr); voteInfo != nil {
		balance, err := v.chain.GetAccountBalanceByTokenId(&addr, &ledger.ViteTokenId)
		if err != nil {
			return nil, err
		}
		if abi.IsActiveRegistration(vmContext, voteInfo.NodeName, gid) {
			return &VoteInfo{voteInfo.NodeName, NodeStatusActive, *bigIntToString(balance)}, nil
		} else {
			return &VoteInfo{voteInfo.NodeName, NodeStatusInActive, *bigIntToString(balance)}, nil

		}
	}
	return nil, nil
}
