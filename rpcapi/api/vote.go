package api

import (
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/ledger/consensus"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
)

type VoteApi struct {
	chain chain.Chain
	cs    consensus.Consensus
	log   log15.Logger
}

func NewVoteApi(vite *vite.Vite) *VoteApi {
	return &VoteApi{
		chain: vite.Chain(),
		cs:    vite.Consensus(),
		log:   log15.New("module", "rpc_api/vote_api"),
	}
}

func (v VoteApi) String() string {
	return "VoteApi"
}

// Private
func (v *VoteApi) GetVoteData(gid types.Gid, name string) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameVote, gid, name)
}

// Private
func (v *VoteApi) GetCancelVoteData(gid types.Gid) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameCancelVote, gid)
}

var (
	NodeStatusActive   uint8 = 1
	NodeStatusInActive uint8 = 2
)
