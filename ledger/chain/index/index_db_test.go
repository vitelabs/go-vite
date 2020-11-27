package chain_index

import (
	"path"
	"testing"

	"gotest.tools/assert"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
)

func TestDumpFileLocation(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewIndexDB(chainDir)
	assert.NilError(t, err)
	step := uint64(75 * 10)
	from := types.GenesisHeight

	for i := from; ; i = i + step {
		location, err := db.GetSnapshotBlockLocation(i)
		assert.NilError(t, err)
		if location == nil {
			break
		}
		t.Log(location.FileId, location.Offset)
	}
}

func TestIndexDB_GetLatestAccountBlock(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewIndexDB(chainDir)
	assert.NilError(t, err)
	address, err := types.HexToAddress("vite_cfe352f1778a77d6ee685182f2d15c4c4c1367ea612593171c")
	assert.NilError(t, err)
	height, location, err := db.GetLatestAccountBlock(&address)
	assert.NilError(t, err)

	t.Log(height, location.FileId, location.Offset)
}
