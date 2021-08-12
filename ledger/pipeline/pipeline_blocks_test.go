package pipeline

import (
	"testing"
	"time"

	chain_block "github.com/vitelabs/go-vite/ledger/chain/block"
	"gotest.tools/assert"
)

func TestPipeline(t *testing.T) {
	prepareTestData(t, tmpDir)

	pipeline, err := newBlocksPipelineWithRun(tmpDir, 400, testFilesize)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	time.Sleep(time.Second)

	chunk1 := pipeline.Peek()
	chunk2 := pipeline.Peek()
	assert.Equal(t, len(chunk1.SnapshotChunks), len(chunk2.SnapshotChunks))
	assert.Equal(t, chunk1.SnapshotRange[0].Height, chunk2.SnapshotRange[0].Height)
	assert.Equal(t, chunk1.SnapshotRange[1].Height, chunk2.SnapshotRange[1].Height)
	t.Log(chunk1.SnapshotRange[0].Height, chunk1.SnapshotRange[1].Height)
}

func TestPipeline2(t *testing.T) {
	// prepareTestData(t, tmpDir)

	pipeline, err := newBlocksPipelineWithRun("/var/folders/d9/r2nnrbfd5pj2390fv10v88sc0000gn/T/868980096", 1, chain_block.FixFileSize)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	for pipeline.Peek() == nil {
		time.Sleep(time.Second)
	}
	chunk1 := pipeline.Peek()
	chunk2 := pipeline.Peek()
	assert.Equal(t, len(chunk1.SnapshotChunks), len(chunk2.SnapshotChunks))
	assert.Equal(t, chunk1.SnapshotRange[0].Height, chunk2.SnapshotRange[0].Height)
	assert.Equal(t, chunk1.SnapshotRange[1].Height, chunk2.SnapshotRange[1].Height)
	t.Log(chunk1.SnapshotRange[0].Height, chunk1.SnapshotRange[1].Height)
}
