package chain_block

import (
	"io"
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

	start := chain_file_manager.NewLocation(1, 0)
	current := start
	for {
		sb, _, nextLocation, err := db.ReadBlock(current)
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
