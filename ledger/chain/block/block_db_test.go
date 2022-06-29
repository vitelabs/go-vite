package chain_block

import (
	"io"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/crypto"
	chain_file_manager "github.com/vitelabs/go-vite/v2/ledger/chain/file_manager"
)

func TestReadSnapshotBlocks(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect ledger data.")

	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger_2101_2")
	db, err := NewBlockDB(chainDir)
	assert.NoError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	start := chain_file_manager.NewLocation(1, 0)
	current := start
	i := 0
	for i < 10 {
		sb, _, nextLocation, err := db.ReadUnit(current)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		if nextLocation == nil {
			break
		}
		t.Log("location", nextLocation.String())
		if sb != nil {
			i++
			t.Log(sb.Height, sb.Hash, current.String())
		}
		current = nextLocation
	}
}

func TestReadAccountBlocks(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect ledger data.")

	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger_2101_2")
	db, err := NewBlockDB(chainDir)
	assert.NoError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	i := 0
	start := chain_file_manager.NewLocation(1, 0)
	current := start
	for i < 100 {
		_, ab, nextLocation, err := db.ReadUnit(current)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		if nextLocation == nil {
			break
		}
		if ab != nil {
			i++
			t.Log(ab.AccountAddress, ab.Height, ab.Hash)
		}
		current = nextLocation
	}
}

func TestReadLocation(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect ledger data.")

	chainDir := path.Join(common.HomeDir(), ".gvite/mockdata/ledger_2101_1")
	db, err := NewBlockDB(chainDir)
	assert.NoError(t, err)
	statusList := db.GetStatus()
	for _, status := range statusList {
		t.Log(status.Name, status.Count, status.Size, status.Status)
	}

	location := chain_file_manager.NewLocation(1, 7333187)
	printLocationContext(t, location, db)
}

func TestDiffBlocksDB(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect ledger data.")

	chainDirA := path.Join(common.HomeDir(), ".gvite/mockdata/ledger_2101_1")
	chainDirB := path.Join(common.HomeDir(), ".gvite/mockdata/ledger_2101_2")
	dbA, err := NewBlockDB(chainDirA)
	assert.NoError(t, err)
	dbB, err := NewBlockDB(chainDirB)
	assert.NoError(t, err)

	location := chain_file_manager.NewLocation(1, 7333187)
	//printLocationContext(t, location, dbA)
	//printLocationContext(t, location, dbB)

	for {
		bytA, nextLocationA, err := dbA.ReadUnitBytes(location)
		assert.NoError(t, err)
		bytB, nextLocationB, err := dbB.ReadUnitBytes(location)
		assert.NoError(t, err)

		t.Log(location.FileId, location.Offset)
		assert.Equal(t, nextLocationA.FileId, nextLocationB.FileId, nextLocationA.String(), nextLocationB.String(), location.String())
		assert.Equal(t, nextLocationA.Offset, nextLocationB.Offset, nextLocationA.String(), nextLocationB.String(), location.String())
		assert.Equal(t, len(bytA), len(bytB))
		assert.EqualValues(t, crypto.Hash256(bytA), crypto.Hash256(bytB))

		location = nextLocationA
	}
}

func printLocationContext(t *testing.T, location *chain_file_manager.Location, db *BlockDB) {
	sb, ab, nextLocation, err := db.ReadUnit(location)
	assert.NoError(t, err)
	if sb != nil {
		t.Log("snapshot block", sb.Height, sb.Hash)
	}
	if ab != nil {
		t.Log("account block", ab.AccountAddress, ab.Height, ab.Hash)
		//byt, _ := json.Marshal(ab)
		//t.Log(string(byt))
	}
	if nextLocation != nil {
		t.Log("next location", nextLocation.FileId, nextLocation.Offset)
	}
}
