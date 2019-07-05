package client

import (
	"strconv"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

//{
//"jsonrpc": "2.0",
//"id": 17,
//"method": "register_getRewardByIndex",
//"params": ["00000000000000000001","0"]
//}

func (c *rpcClient) GetRewardByIndex(index uint64) (reward *api.RewardInfo, err error) {
	err = c.cc.Call(&reward, "register_getRewardByIndex", types.SNAPSHOT_GID, strconv.FormatUint(index, 10))
	return
}

//{
//"jsonrpc": "2.0",
//"id": 17,
//"method":"vote_getVoteDetails",
//"params":[0]
//}

func (c *rpcClient) GetVoteDetailsByIndex(index uint64) (details []*consensus.VoteDetails, err error) {
	err = c.cc.Call(&details, "vote_getVoteDetails", index)
	return
}
