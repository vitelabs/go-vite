package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_db"
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
	return abi.ABIConsensusGroup.PackMethod(abi.MethodNameVote, gid, name)
}
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

func (v *VoteApi) GetVoteInfo(gid types.Gid, addr types.Address) (*VoteInfo, error) {
	// TODO tmpchain
	var tmpChain vm_db.Chain
	prevHash, err := getPrevBlockHash(v.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(tmpChain, &types.AddressConsensusGroup, &v.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	voteInfo, err := abi.GetVote(db, gid, addr)
	if err != nil {
		return nil, err
	}
	if voteInfo != nil {
		balance, err := tmpChain.GetBalance(addr, ledger.ViteTokenId)
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
