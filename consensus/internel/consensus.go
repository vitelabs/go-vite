package internel

import (
	"math/big"

	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type Detail struct {
	PlanNum   uint64 // member plan cnt
	ActualNum uint64 // member actual cnt
	//top100, index: nodeName: balance
	PeriodM map[uint64]*PeriodDetails
}

type PeriodDetails struct {
	ActualNum uint64 // actual block num in period
	VoteMap   map[string]*big.Int
}

type ConsensusReader interface {
	PeriodTime(info *types.ConsensusGroupInfo) (uint64, error)
	TimeToIndex(time, genesisTime time.Time, info *types.ConsensusGroupInfo) (uint64, error)
	// return
	Detail(startIndex, endIndex uint64, info *types.ConsensusGroupInfo, register *types.Registration, database vmctxt_interface.VmDatabase) (*Detail, error)
}

func NewReader(info *types.ConsensusGroupInfo) ConsensusReader {
	return nil
}
