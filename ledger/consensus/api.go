package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/cdb"
)

// APISnapshot is the interface that can query snapshot consensus info.
type APISnapshot struct {
	snapshot *snapshotCs
}

// ReadVoteMap query the vote result by time.
func (api *APISnapshot) ReadVoteMap(ti time.Time) ([]*VoteDetails, *ledger.HashHeight, error) {
	return api.snapshot.voteDetailsBeforeTime(ti)
}

// ReadSuccessRate query success rate for every SBP.
func (api *APISnapshot) ReadSuccessRate(start, end uint64) ([]map[types.Address]*cdb.Content, error) {
	var result []map[types.Address]*cdb.Content
	for i := start; i < end; i++ {
		rateByHour, err := api.snapshot.rw.GetSuccessRateByHour2(i)
		if err != nil {
			panic(err)
		}
		result = append(result, rateByHour)
	}
	return result, nil
}

func (api APISnapshot) ReadByIndex(gid types.Gid, index uint64) ([]*Event, uint64, error) {
	// cal votes
	eResult, err := api.snapshot.ElectionIndex(index)
	if err != nil {
		return nil, 0, err
	}

	voteTime := api.snapshot.GenProofTime(index)
	var result []*Event
	for _, p := range eResult.Plans {
		e := newConsensusEvent(eResult, p, gid, voteTime)
		result = append(result, &e)
	}
	return result, uint64(eResult.Index), nil
}
