package internel

import (
	"math/big"

	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type Detail struct {
	PlanNum   uint64
	ActualNum uint64
	//top100, index: nodeName: balance
	VoteM map[uint64]map[string]*big.Int
}

func PeriodTime(info *types.ConsensusGroupInfo) (uint64, error) {
	return 0, nil
}
func TimeToIndex(time, genesisTime time.Time, info *types.ConsensusGroupInfo) (uint64, error) {
	return 0, nil
}

// return
func GetDetail(startIndex, endIndex uint64, info *types.ConsensusGroupInfo, register *types.Registration, database vmctxt_interface.VmDatabase) (*Detail, error) {
	return nil, nil
}
