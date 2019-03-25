package sync_cache

import (
	"io"

	"github.com/vitelabs/go-vite/ledger"
)

type ReadCloser interface {
	// Read a block, return io.EOF if reach end, the block maybe a accountBlock or a snapshotBlock
	Read() (accountBlock *ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock, err error)
	// Close the stream
	Close() error
}

type SyncCache interface {
	NewWriter(from, to uint64) (io.WriteCloser, error)
	Chunks() SegmentList
	NewReader(from, to uint64) (ReadCloser, error)
}
