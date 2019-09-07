package vm_db

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (vdb *vmDb) AddLog(log *ledger.VmLog) {
	vdb.unsaved().AddLog(log)
}

func (vdb *vmDb) GetLogList() ledger.VmLogList {
	return vdb.unsaved().GetLogList()
}

func (vdb *vmDb) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return vdb.chain.GetVmLogList(logHash)
}

func (vdb *vmDb) GetLogListHash() *types.Hash {
	var sbHeight uint64
	if !vdb.isGenesis {
		latestSb, err := vdb.LatestSnapshotBlock()
		if err != nil {
			panic(fmt.Sprintf("Error: %s", err.Error()))
		}
		sbHeight = latestSb.Height
	}

	return vdb.unsaved().GetLogListHash(sbHeight, *vdb.Address(), vdb.PrevAccountBlockHash())
}
