package internel

import (
	"math/big"

	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type Detail struct {
	planNum   uint64
	actualNum uint64
	//top100, index: nodeName: balance
	voteM map[uint64]map[string]*big.Int
}

type ConsensusReader interface {
	PeriodTime(info *types.ConsensusGroupInfo) (uint64, error)
	TimeToIndex(time, genesisTime time.Time, info *types.ConsensusGroupInfo) (uint64, error)
	// return
	Detail(startIndex, endIndex uint64, info *types.ConsensusGroupInfo, register *types.Registration, database vmctxt_interface.VmDatabase) (*Detail, error)
}
