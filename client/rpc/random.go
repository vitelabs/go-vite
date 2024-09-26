package rpc

import (
	"strconv"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
	"github.com/vitelabs/go-vite/v2/rpc"
	"github.com/vitelabs/go-vite/v2/rpcapi/api"
)

// OnroadApi ...
type RandomApi interface {
	GetRewardByIndex(index uint64) (reward *api.RewardInfo, err error)
	GetVoteDetailsByIndex(index uint64) (details []*consensus.VoteDetails, err error)
	GetRewardPendingWithdrawal(sbpName string) (reward *api.SBPReward, err error)
	GetCurrentCycle() (index uint64, err error)
	RawCall(method string, params ...interface{}) (interface{}, error)
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

func (c *randomApi) GetRewardPendingWithdrawal(sbpName string) (reward *api.SBPReward, err error) {
	err = c.cc.Call(&reward, "contract_getSBPRewardPendingWithdrawal", sbpName)
	return
}

func (c *randomApi) GetCurrentCycle() (result uint64, err error) {
	err = c.cc.Call(&result, "dex_getPeriodId")
	return
}

func (c *randomApi) RawCall(method string, params ...interface{}) (result interface{}, err error) {
	err = c.cc.Call(&result, method, params...)
	return
}
