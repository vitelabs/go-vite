package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
