package chain_block

import (
	"path"
	"testing"

	"gotest.tools/assert"

	"github.com/vitelabs/go-vite/common"
	chain_file_manager "github.com/vitelabs/go-vite/ledger/chain/file_manager"
)

func TestRead(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewBlockDB(chainDir)
	assert.NilError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	locations := []*chain_file_manager.Location{
		chain_file_manager.NewLocation(1, 489),
		chain_file_manager.NewLocation(1, 355741),
		chain_file_manager.NewLocation(1, 533089),
		chain_file_manager.NewLocation(1, 710445),
		chain_file_manager.NewLocation(1, 887812),
	}

	for i := 0; i < len(locations)-1; i++ {
		blocks, err := db.ReadRange(locations[i], locations[i+1])
		assert.NilError(t, err)
		for _, block := range blocks {
			sblock := block.SnapshotBlock
			t.Log(sblock.Height, sblock.Hash)
		}
	}
}
