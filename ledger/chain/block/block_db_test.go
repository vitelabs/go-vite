package chain_block

import (
	"io"
	"path"
	"testing"

	"gotest.tools/assert"

	"github.com/vitelabs/go-vite/common"
	chain_file_manager "github.com/vitelabs/go-vite/ledger/chain/file_manager"
)

func TestReadSnapshotBlocks(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewBlockDB(chainDir)
	assert.NilError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	start := chain_file_manager.NewLocation(1, 0)
	current := start
	for {
		sb, _, nextLocation, err := db.ReadUnit(current)
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		if nextLocation == nil {
			break
		}
		if sb != nil {
			t.Log(sb.Height, sb.Hash)
		}
		current = nextLocation
	}
}

func TestReadAccountBlocks(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewBlockDB(chainDir)
	assert.NilError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	start := chain_file_manager.NewLocation(1, 0)
	current := start
	for {
		_, ab, nextLocation, err := db.ReadUnit(current)
		if err == io.EOF {
			break
		}
		assert.NilError(t, err)
		if nextLocation == nil {
			break
		}
		if ab != nil {
			t.Log(ab.AccountAddress, ab.Height, ab.Hash)
		}
		current = nextLocation
	}
}

func TestReadLocation(t *testing.T) {
	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger")
	db, err := NewBlockDB(chainDir)
	assert.NilError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	location := chain_file_manager.NewLocation(1, 0)
	sb, ab, nextLocation, err := db.ReadUnit(location)
	assert.NilError(t, err)
	if sb != nil {
		t.Log(sb.Height, sb.Hash)
	}
	if ab != nil {
		t.Log(ab.AccountAddress, ab.Height, ab.Hash)
	}
	if nextLocation != nil {
		t.Log(nextLocation.FileId, nextLocation.Offset)
	}
}
