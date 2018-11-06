package api

import (
	"math/big"
	"time"

	"runtime/debug"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vite/net"
)

type DebugApi struct {
	v *vite.Vite
}

func (api DebugApi) Free() {
	debug.FreeOSMemory()
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
func (api DebugApi) ConsensusPlanAndActual(gid types.Gid, offset int64, index uint64) map[string]interface{} {
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
	type PlanActual struct {
		T time.Time
		B ledger.SnapshotBlock
		E consensus.Event
		R bool
	}

	i := big.NewInt(0).SetUint64(index)
	finalIndex := i.Add(i, big.NewInt(offset))

	var blocks []*ledger.SnapshotBlock
	stime, etime, err := api.v.Consensus().VoteIndexToTime(gid, finalIndex.Uint64())
	block, err := api.v.Chain().GetSnapshotBlockBeforeTime(etime)
	if err != nil {
		result["err"] = err
		return result
	}
	blocks = append(blocks, block)

	nextHeight := block.Height - 1
	for block.Height > types.EmptyHeight {
		b, err := api.v.Chain().GetSnapshotBlockByHeight(nextHeight)
		if err != nil {
			result["err"] = err
			result["blocks"] = blocks
			return result
		}
		if b == nil {
			break
		}
		if b.Timestamp.Before(*stime) {
			break
		}
		blocks = append(blocks, b)
		nextHeight = b.Height - 1
	}

	result["blocks"] = blocks

	events, resultIndex, err := api.v.Consensus().ReadByIndex(gid, finalIndex.Uint64())
	if err != nil {
		result["err"] = err
		return result
	}
	result["events"] = events
	result["index"] = finalIndex
	result["rIndex"] = resultIndex
	result["stime"] = stime
	result["etime"] = etime

	merge := make(map[time.Time]*PlanActual)

	for _, v := range events {
		merge[v.Timestamp] = &PlanActual{E: *v, T: v.Timestamp}
	}

	for _, v := range blocks {
		a, ok := merge[*v.Timestamp]
		if ok {
			a.B = *v
			if v.Producer() == a.E.Address {
				a.R = true
			}
		} else {
			merge[*v.Timestamp] = &PlanActual{B: *v, T: *v.Timestamp}
		}
	}
	result["merge"] = merge
	return result
}

func (api DebugApi) FetchTaskQueue() []*net.Task {
	return api.v.Net().Tasks()
}

func (api DebugApi) MachineInfo() map[string]interface{} {
	result := make(map[string]interface{})
	result["now"] = time.Now().String()
	return result
}

func NewDebugApi(v *vite.Vite) *DebugApi {
	return &DebugApi{
		v: v,
	}
}
