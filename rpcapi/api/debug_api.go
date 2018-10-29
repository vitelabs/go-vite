package api

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite"
)

type DebugApi struct {
	v *vite.Vite
}

func (api DebugApi) PoolInfo(addr *types.Address) string {
	return api.v.Pool().Info(addr)
}

func (api DebugApi) PoolSnapshot() map[string]interface{} {
	return api.v.Pool().Snapshot()
}

func (api DebugApi) PoolAccount(addr types.Address) map[string]interface{} {
	return api.v.Pool().Account(addr)
}

func (api DebugApi) PoolSnapshotChainDetail(chainId string) map[string]interface{} {
	return api.v.Pool().SnapshotChainDetail(chainId)
}

func (api DebugApi) PoolAccountChainDetail(addr types.Address, chainId string) map[string]interface{} {
	return api.v.Pool().AccountChainDetail(addr, chainId)
}

func (api DebugApi) P2pNodes() []string {
	if p2p := api.v.P2P(); p2p != nil {
		return p2p.Nodes()
	}

	return nil
}

func (api DebugApi) ConsensusProducers(gid types.Gid, offset int64, index uint64) map[string]interface{} {
	result := make(map[string]interface{})
	if index == 0 {
		head := api.v.Chain().GetLatestSnapshotBlock()
		i, e := api.v.Consensus().VoteTimeToIndex(gid, *head.Timestamp)
		if e != nil {
			result["err"] = e
			return result
		}
		index = i
	}

	i := big.NewInt(0).SetUint64(index)
	finalIndex := i.Add(i, big.NewInt(offset))

	events, u, e := api.v.Consensus().ReadByIndex(gid, finalIndex.Uint64())
	if e != nil {
		result["err"] = e
		return result
	}

	result["events"] = events
	result["index"] = u
	return result
}

func (api DebugApi) ConsensusVoteDetails(gid types.Gid, offset int64, index uint64) map[string]interface{} {
	result := make(map[string]interface{})
	if index == 0 {
		head := api.v.Chain().GetLatestSnapshotBlock()
		i, e := api.v.Consensus().VoteTimeToIndex(gid, *head.Timestamp)
		if e != nil {
			result["err"] = e
			return result
		}
		index = i
	}

	i := big.NewInt(0).SetUint64(index)
	finalIndex := i.Add(i, big.NewInt(offset))

	events, u, e := api.v.Consensus().ReadVoteMapByTime(gid, finalIndex.Uint64())
	if e != nil {
		result["err"] = e
		return result
	}

	result["events"] = events
	result["hashH"] = u
	return result
}

func NewDebugApi(v *vite.Vite) *DebugApi {
	return &DebugApi{
		v: v,
	}
}
