package api

import (
	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
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

type VoteInfo struct {
	Name       string `json:"nodeName"`
	NodeStatus uint8  `json:"nodeStatus"`
	Balance    string `json:"balance"`
}

// Deprecated: use contract_getVotedSBP instead
func (v *VoteApi) GetVoteInfo(gid types.Gid, addr types.Address) (*VoteInfo, error) {
	db, err := getVmDb(v.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	voteInfo, err := abi.GetVote(db, gid, addr)
	if err != nil {
		return nil, err
	}
	if voteInfo != nil {
		balance, err := v.chain.GetBalance(addr, ledger.ViteTokenId)
		if err != nil {
			return nil, err
		}
		active, err := abi.IsActiveRegistration(db, voteInfo.SbpName, gid)
		if err != nil {
			return nil, err
		}
		if active {
			return &VoteInfo{voteInfo.SbpName, NodeStatusActive, *bigIntToString(balance)}, nil
		} else {
			return &VoteInfo{voteInfo.SbpName, NodeStatusInActive, *bigIntToString(balance)}, nil
		}
	}
	return nil, nil
}

// Deprecated: use contract_getSBPVotingDetailsByCycle instead
func (v *VoteApi) GetVoteDetails(index *uint64) ([]*consensus.VoteDetails, error) {
	t := time.Now()
	if index != nil {
		_, etime := v.cs.SBPReader().GetDayTimeIndex().Index2Time(*index)
		t = etime
	}

	details, _, err := v.cs.API().ReadVoteMap(t)
	if err != nil {
		return nil, err
	}
	return details, nil
}
