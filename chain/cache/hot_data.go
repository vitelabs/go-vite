package chain_cache

import "github.com/vitelabs/go-vite/ledger"

type hotData struct {
	ds *dataSet

	latestSbDataId  uint64
	genesisSbDataId uint64
}

func newHotData(ds *dataSet) *hotData {
	return &hotData{
		ds: ds,
	}
}

func (hd *hotData) UpdateLatestSnapshotBlock(latestSbDataId uint64) {
	if hd.latestSbDataId == latestSbDataId {
		return
	}
	if hd.latestSbDataId > 0 {
		hd.ds.UnRefDataId(hd.latestSbDataId)
	}

	hd.latestSbDataId = latestSbDataId
	hd.ds.RefDataId(latestSbDataId)

}

func (hd *hotData) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	if hd.latestSbDataId <= 0 {
		return nil
	}

	return hd.ds.GetSnapshotBlock(hd.latestSbDataId)
}

func (hd *hotData) SetGenesisSnapshotBlock(dataId uint64) {
	hd.genesisSbDataId = dataId
	hd.ds.RefDataId(dataId)
}

func (hd *hotData) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return hd.ds.GetSnapshotBlock(hd.genesisSbDataId)
}
