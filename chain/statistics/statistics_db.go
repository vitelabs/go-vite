package statistics

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type StatisticsDB struct {
	chain chain.Chain
	db    *leveldb.DB

	targetSnapshotHeight uint64
	// listen snapshot insert and rollback
	subscribePendingCache []*ledger.SnapshotBlock

	log log15.Logger
}

func NewStatisticsDB(chain chain.Chain) *StatisticsDB {
	db, err := chain.NewDb("statistics_db")
	if err != nil {
		panic(err)
	}
	statisticsDB := &StatisticsDB{
		chain:                chain,
		db:                   db,
		targetSnapshotHeight: 2,
		log:                  log15.New("module", "StatisticsDB"),
	}
	if latestSnapshot := chain.GetLatestSnapshotBlock(); latestSnapshot != nil {
		statisticsDB.targetSnapshotHeight = latestSnapshot.Height
	}
	return statisticsDB
}

func (sDB *StatisticsDB) Close() error {
	if err := sDB.db.Close(); err != nil {
		return errors.New(fmt.Sprintf("sDB.db.Close failed, error is %s", err.Error()))
	}
	return nil
}

func (sDB *StatisticsDB) Reconstruct() {
}

func (sDB *StatisticsDB) chaseTargetSnapshotHeight(currentHeight, targetHeight uint64) {

}

func (sDB *StatisticsDB) GetOnroadAccountInfo(addr types.Address) (*OnroadAccountInfo, error) {
	omMap, err := sDB.getOnroadInfo(addr)
	if err != nil {
		return nil, err
	}
	onroadInfo := &OnroadAccountInfo{
		AccountAddress:      addr,
		TotalNumber:         0,
		TokenBalanceInfoMap: make(map[types.TokenTypeId]*TokenBalanceInfo),
	}
	balanceMap := onroadInfo.TokenBalanceInfoMap
	for k, v := range omMap {
		balanceMap[k] = &TokenBalanceInfo{
			TotalAmount: v.TotalAmount,
			Number:      v.Number,
		}
		onroadInfo.TotalNumber += v.Number
	}
	return onroadInfo, nil
}
