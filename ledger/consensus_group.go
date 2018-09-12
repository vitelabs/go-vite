package ledger

import "github.com/vitelabs/go-vite/common/types"

type ConsensusGroup struct {
	nodeCount              uint8
	interval               int64
	countingRuleId         uint8
	countingRuleParam      []byte
	registerConditionId    uint8
	registerConditionParam []byte
	voteConditionId        uint8
	voteConditionParam     []byte
}

func CommonGid() *types.Gid {
	return nil
}
