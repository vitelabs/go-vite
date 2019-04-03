package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/consensus/db"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type ApiSnapshot struct {
	snapshot *snapshotCs
}

//
//func (self *ApiSnapshot) ReadSnapshotVoteMapByTime(gid types.Gid, index uint64) ([]*VoteDetails, *ledger.HashHeight, error) {
//	t, ok := self.tellers.Load(gid)
//	if !ok {
//		tmp, err := self.initTeller(gid)
//		if err != nil {
//			return nil, nil, err
//		}
//		t = tmp
//	}
//	if t == nil {
//		return nil, nil, errors.New("consensus group not exist")
//	}
//	tel := t.(*teller)
//
//	return tel.voteDetails(index)
//}
//
func (self *ApiSnapshot) ReadVoteMap(ti time.Time) ([]*VoteDetails, *ledger.HashHeight, error) {
	return self.snapshot.voteDetailsBeforeTime(ti)
}

func (self *ApiSnapshot) ReadSuccessRateForAPI(start, end uint64) ([]map[types.Address]*consensus_db.Content, error) {
	var result []map[types.Address]*consensus_db.Content
	for i := start; i < end; i++ {
		rateByHour, err := self.snapshot.rw.GetSuccessRateByHour2(i)
		if err != nil {
			panic(err)
		}
		result = append(result, rateByHour)
	}
	return result, nil
}

//
//func (self *ApiSnapshot) ReadSuccessRate2ForAPI(start, end uint64) ([]SBPInfos, error) {
//	var result []SBPInfos
//	for i := start; i < end; i++ {
//		rateByHour, err := self.periods.GetByHeight(i)
//		if err != nil {
//			panic(err)
//		}
//		point := rateByHour.(*periodPoint)
//		result = append(result, point.GetSBPInfos())
//	}
//	return result, nil
//}
