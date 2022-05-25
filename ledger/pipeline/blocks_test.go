package pipeline

import (
	"os"
	"testing"

	"github.com/vitelabs/go-vite/v2/common/fileutils"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
	chain_block "github.com/vitelabs/go-vite/v2/ledger/chain/block"
)

func prepareTestData(t *testing.T, dir string) {
	// prepare once
	if _, err := os.Stat(dir + "/lock"); os.IsNotExist(err) {
		// os.RemoveAll(dir)
		os.Create(dir + "/lock")
	} else {
		return
	}

	blockDb, err := chain_block.NewBlockDBFixedSize(dir, testFilesize)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer blockDb.Close()

	len := 3000
	for i := 0; i < len; i++ {
		var block core.SnapshotBlock
		block.Mock(uint64(i))

		chunk := &core.SnapshotChunk{}
		chunk.SnapshotBlock = &block
		_, _, err = blockDb.Write(chunk)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	blockDb.Prepare()
	blockDb.Commit()

	t.Log("prepare data done", dir)
}

var (
	tmpDir       = fileutils.CreateTempDir()
	testFilesize = int64(5 * 1024)
)

func TestNewBlocks(t *testing.T) {
	prepareTestData(t, tmpDir)
	_, err := newBlocks(tmpDir, testFilesize)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
