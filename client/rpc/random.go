package rpc

import (
	"strconv"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

// OnroadApi ...
type RandomApi interface {
	GetRewardByIndex(index uint64) (reward *api.RewardInfo, err error)
	GetVoteDetailsByIndex(index uint64) (details []*consensus.VoteDetails, err error)
}

type randomApi struct {
	cc *rpc.Client
}

func NewRandomApi(cc *rpc.Client) RandomApi {
	return &randomApi{cc: cc}
}

//{
//"jsonrpc": "2.0",
//"id": 17,
//"method": "register_getRewardByIndex",
//"params": ["00000000000000000001","0"]
//}
func (c *randomApi) GetRewardByIndex(index uint64) (reward *api.RewardInfo, err error) {
	err = c.cc.Call(&reward, "register_getRewardByIndex", types.SNAPSHOT_GID, strconv.FormatUint(index, 10))
	return
}

//{
//"jsonrpc": "2.0",
//"id": 17,
//"method":"vote_getVoteDetails",
//"params":[0]
//}
func (c *randomApi) GetVoteDetailsByIndex(index uint64) (details []*consensus.VoteDetails, err error) {
	err = c.cc.Call(&details, "vote_getVoteDetails", index)
	return
}
