package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
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
	return contracts.ABIVote.PackMethod(contracts.MethodNameVote, gid, name)
}
func (v *VoteApi) GetCancelVoteData(gid types.Gid) ([]byte, error) {
	return contracts.ABIVote.PackMethod(contracts.MethodNameCancelVote, gid)
}

type VoteInfo struct {
	Name    string `json:"name"`
	Balance string `json:"balance"`
}

func (v *VoteApi) GetVoteInfo(gid types.Gid, addr types.Address) (*VoteInfo, error) {
	vmContext, err := vm_context.NewVmContext(v.chain, nil, nil, &addr)
	if err != nil {
		return nil, err
	}
	if voteInfo := contracts.GetVote(vmContext, gid, addr); voteInfo != nil {
		balance, err := v.chain.GetAccountBalanceByTokenId(&addr, &ledger.ViteTokenId)
		if err != nil {
			return nil, err
		}
		return &VoteInfo{voteInfo.NodeName, *bigIntToString(balance)}, nil
	}
	return nil, nil
}
