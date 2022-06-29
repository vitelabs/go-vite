package chain_index

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
)

func TestDumpFileLocation(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewIndexDB(chainDir)
	assert.NoError(t, err)
	step := uint64(75 * 10)
	from := types.GenesisHeight

	for i := from; ; i = i + step {
		location, err := db.GetSnapshotBlockLocation(i)
		assert.NoError(t, err)
		if location == nil {
			break
		}
		t.Log(location.FileId, location.Offset)
	}
}

func TestIndexDB_GetLatestAccountBlock(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect IndexDB.")

	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewIndexDB(chainDir)
	assert.NoError(t, err)
	address, err := types.HexToAddress("vite_7c8c9e1e878e8a6ddf59c66a83791a5755a8fcf606c4bd31ea")
	assert.NoError(t, err)
	height, location, err := db.GetLatestAccountBlock(&address)
	assert.NoError(t, err)
	assert.NotNil(t, location)

	t.Log(height, location.FileId, location.Offset)
}
