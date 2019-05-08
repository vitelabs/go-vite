package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/consensus/db"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
func (api *APISnapshot) ReadSuccessRate(start, end uint64) ([]map[types.Address]*consensus_db.Content, error) {
	var result []map[types.Address]*consensus_db.Content
	for i := start; i < end; i++ {
		rateByHour, err := api.snapshot.rw.GetSuccessRateByHour2(i)
		if err != nil {
			panic(err)
		}
		result = append(result, rateByHour)
	}
	return result, nil
}
