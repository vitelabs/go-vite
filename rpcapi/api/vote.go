package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
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
