package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"strings"
	"time"
)

const (
	jsonVote = `
	[
		{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeName","type":"string"}]},
		{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
		{"type":"variable","name":"voteStatus","inputs":[{"name":"nodeName","type":"string"}]}
	]`

	MethodNameVote         = "Vote"
	MethodNameCancelVote   = "CancelVote"
	VariableNameVoteStatus = "voteStatus"
)

var (
	ABIVote, _ = abi.JSONToABIContract(strings.NewReader(jsonVote))
)

type ParamVote struct {
	Gid      types.Gid
	NodeName string
}

func GetVoteKey(addr types.Address, gid types.Gid) []byte {
	return append(gid.Bytes(), addr.Bytes()...)
}

func GetAddrFromVoteKey(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[types.GidSize:])
	return addr
}

func GetVote(db StorageDatabase, gid types.Gid, addr types.Address) *types.VoteInfo {
	defer monitor.LogTime("vm", "GetVote", time.Now())
	data := db.GetStorageBySnapshotHash(&types.AddressConsensusGroup, GetVoteKey(addr, gid), nil)
	if len(data) > 0 {
		nodeName := new(string)
		ABIVote.UnpackVariable(nodeName, VariableNameVoteStatus, data)
		return &types.VoteInfo{addr, *nodeName}
	}
	return nil
}

func GetVoteList(db StorageDatabase, gid types.Gid, snapshotHash *types.Hash) []*types.VoteInfo {
	defer monitor.LogTime("vm", "GetVoteList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, types.SNAPSHOT_GID.Bytes(), snapshotHash)
	} else {
		iterator = db.NewStorageIteratorBySnapshotHash(&types.AddressConsensusGroup, gid.Bytes(), snapshotHash)
	}
	voteInfoList := make([]*types.VoteInfo, 0)
	if iterator == nil {
		return voteInfoList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		voterAddr := GetAddrFromVoteKey(key)
		nodeName := new(string)
		if err := ABIVote.UnpackVariable(nodeName, VariableNameVoteStatus, value); err == nil {
			voteInfoList = append(voteInfoList, &types.VoteInfo{voterAddr, *nodeName})
		}
	}
	return voteInfoList
}
