package api

import (
	"encoding/json"
	"errors"
	"math/big"
	"runtime/debug"
	"time"

	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vite"
)

type DebugApi struct {
	v *vite.Vite
}

func (api DebugApi) Free() {
	debug.FreeOSMemory()
}

func (api DebugApi) PoolInfo() map[string]interface{} {
	return api.v.Pool().Info()
}

func (api DebugApi) PoolSnapshot() map[string]interface{} {
	return api.v.Pool().Snapshot()
}

func (api DebugApi) PoolSnapshotBlockDetail(hash types.Hash) map[string]interface{} {
	info := api.v.Pool().SnapshotBlockInfo(hash)
	m := make(map[string]interface{})
	m["block"] = info
	return m
}

func (api DebugApi) PoolAccount(addr types.Address) map[string]interface{} {
	return api.v.Pool().Account(addr)
}

func (api DebugApi) PoolSnapshotChainDetail(chainId string, height uint64) map[string]interface{} {
	return api.v.Pool().SnapshotChainDetail(chainId, height)
}

func (api DebugApi) PoolAccountChainDetail(addr types.Address, chainId string, height uint64) map[string]interface{} {
	return api.v.Pool().AccountChainDetail(addr, chainId, height)
}

func (api DebugApi) PoolAccountBlockDetail(addr types.Address, hash types.Hash) map[string]interface{} {
	info := api.v.Pool().AccountBlockInfo(addr, hash)
	m := make(map[string]interface{})
	m["block"] = info
	return m
}

func (api DebugApi) PoolIrreversible() string {
	block := api.v.Pool().GetIrreversibleBlock()
	if block == nil {
		return "empty"
	}
	bytes, _ := json.Marshal(block)
	return string(bytes)
}
func (api DebugApi) RoadInfo() map[string]interface{} {
	return api.v.OnRoad().Info()
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

func (api DebugApi) ConsensusBlockRate(gid types.Gid, startIndex, endIndex uint64) map[string]interface{} {
	// todo
	return nil

	//ch := api.v.Chain()
	//genesis := chain.GenesisSnapshotBlock
	//
	//block := ch.GetLatestSnapshotBlock()
	//
	//registers, err := ch.GetRegisterList(block.Hash, gid)
	//if err != nil {
	//	return errMap(err)
	//}
	//infos, err := ch.GetConsensusGroupList(block.Hash)
	//if err != nil {
	//	return errMap(err)
	//}
	//var info *types.ConsensusGroupInfo
	//for _, cs := range infos {
	//	if cs.Gid == gid {
	//		info = cs
	//		break
	//	}
	//}
	//if info == nil {
	//	return errMap(errors.New("can't find group."))
	//}
	//reader := core.NewReader(*genesis.Timestamp, info)
	//u, err := reader.TimeToIndex(*block.Timestamp)
	//if err != nil {
	//	return errMap(err)
	//}
	//if u < endIndex {
	//	endIndex = u
	//}
	//if endIndex <= 0 {
	//	endIndex = u
	//}
	//first, err := ch.GetSnapshotBlockHeadByHeight(3)
	//if err != nil {
	//	return errMap(err)
	//}
	//if first == nil {
	//	return errMap(errors.New("first block is nil."))
	//}
	//fromIndex, err := reader.TimeToIndex(*first.Timestamp)
	//if err != nil {
	//	return errMap(err)
	//}
	//if startIndex < fromIndex {
	//	startIndex = fromIndex
	//}
	//if startIndex <= 0 {
	//	startIndex = fromIndex
	//}
	//type Rate struct {
	//	Actual uint64
	//	Plan   uint64
	//	Rate   uint64
	//}
	//m := make(map[string]interface{})
	//
	//for _, register := range registers {
	//	detail, err := reader.VoteDetails(startIndex, endIndex, register, ch)
	//	if err != nil {
	//		return errMap(err)
	//	}
	//
	//	rate := uint64(0)
	//	if detail.PlanNum > 0 {
	//		rate = (detail.ActualNum * 10000.0) / detail.PlanNum
	//	}
	//	m[register.Name] = &Rate{
	//		Actual: detail.ActualNum,
	//		Plan:   detail.PlanNum,
	//		Rate:   rate,
	//	}
	//}
	//m["startIndex"] = startIndex
	//m["endIndex"] = endIndex
	//s, _, err := api.v.Consensus().VoteIndexToTime(gid, startIndex)
	//if err != nil {
	//	return errMap(err)
	//}
	//m["startTime"] = s.String()
	//e, _, err := api.v.Consensus().VoteIndexToTime(gid, endIndex)
	//if err != nil {
	//	return errMap(err)
	//}
	//m["endTime"] = e.String()
	//return m
}
func errMap(err error) map[string]interface{} {
	m := make(map[string]interface{})
	m["err"] = err
	return m
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

func (api DebugApi) SetGetTestTokenLimitSize(size int) error {
	testtokenlruLimitSize = size
	return nil
}
func (api DebugApi) peersDetails() map[string]interface{} {
	return nil
}

func (api DebugApi) GetForkInfo() config.ForkPoints {
	return fork.GetForkPoints()
}
func (api DebugApi) GetRecentActiveFork() *fork.ForkPointItem {
	return fork.GetRecentActiveFork(api.v.Chain().GetLatestSnapshotBlock().Height)
}

func (api DebugApi) GetOnRoadInfoUnconfirmed(addr types.Address) ([]*types.Hash, error) {
	return api.v.Chain().GetOnRoadInfoUnconfirmedHashList(addr)
}

func (api DebugApi) UpdateOnRoadInfo(addr types.Address, tkId types.TokenTypeId, number uint64, amountStr *string) error {
	amount := big.NewInt(0)
	if amountStr != nil {
		if _, ok := amount.SetString(*amountStr, 10); !ok {
			return ErrStrToBigInt
		}
	}
	result := amount.Cmp(helper.Big0)
	if result < 0 || (number == 0 && result > 0) {
		return errors.New("amount invalid")
	}
	return api.v.Chain().UpdateOnRoadInfo(addr, tkId, number, *amount)
}

func (api DebugApi) ClearOnRoadUnconfirmedCache(addr types.Address, hashList []*types.Hash) error {
	return api.v.Chain().ClearOnRoadUnconfirmedCache(addr, hashList)
}
