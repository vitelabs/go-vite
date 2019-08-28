package api

import (
	"math/big"
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
	return abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, gid, name)
}

// Private
func (v *VoteApi) GetCancelVoteData(gid types.Gid) ([]byte, error) {
	return abi.ABIConsensusGroup.PackMethod(abi.MethodNameCancelVote, gid)
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

// Deprecated: use contract_getSBPVotingInfo instead
func (v *VoteApi) GetVoteInfo(gid types.Gid, addr types.Address) (*VoteInfo, error) {
	db, err := getVmDb(v.chain, types.AddressConsensusGroup)
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
		active, err := abi.IsActiveRegistration(db, voteInfo.NodeName, gid)
		if err != nil {
			return nil, err
		}
		if active {
			return &VoteInfo{voteInfo.NodeName, NodeStatusActive, *bigIntToString(balance)}, nil
		} else {
			return &VoteInfo{voteInfo.NodeName, NodeStatusInActive, *bigIntToString(balance)}, nil
		}
	}
	return nil, nil
}
func (v *ContractApi) GetSBPVotingInfo(gid types.Gid, addr types.Address) (*VoteInfo, error) {
	db, err := getVmDb(v.chain, types.AddressConsensusGroup)
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
		active, err := abi.IsActiveRegistration(db, voteInfo.NodeName, gid)
		if err != nil {
			return nil, err
		}
		if active {
			return &VoteInfo{voteInfo.NodeName, NodeStatusActive, *bigIntToString(balance)}, nil
		} else {
			return &VoteInfo{voteInfo.NodeName, NodeStatusInActive, *bigIntToString(balance)}, nil
		}
	}
	return nil, nil
}

// Deprecated: use contract_getSBPVotingDetailsByIndex instead
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

type VoteDetail struct {
	Name            string                     `json:"name"`
	VoteNum         *big.Int                   `json:"voteNum"`
	CurrentAddr     types.Address              `json:"currentProducerAddress"`
	HistoryAddrList []types.Address            `json:"historyProducerAddresses"`
	VoteMap         map[types.Address]*big.Int `json:"voteMap"`
}

func (v *ContractApi) GetVoteDetails(index *uint64) ([]*VoteDetail, error) {
	t := time.Now()
	if index != nil {
		_, etime := v.cs.SBPReader().GetDayTimeIndex().Index2Time(*index)
		t = etime
	}
	details, _, err := v.cs.API().ReadVoteMap(t)
	if err != nil {
		return nil, err
	}
	list := make([]*VoteDetail, len(details))
	for i, detail := range details {
		list[i] = &VoteDetail{
			Name:            detail.Name,
			VoteNum:         detail.Balance,
			CurrentAddr:     detail.CurrentAddr,
			HistoryAddrList: detail.RegisterList,
			VoteMap:         detail.Addr,
		}
	}
	return list, nil
}
