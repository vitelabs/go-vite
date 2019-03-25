package sync_cache

import (
	"github.com/vitelabs/go-vite/ledger"
	"io"
)

type ReadCloser interface {
	Read(from, to uint64, fn func(accblock []*ledger.AccountBlock, sblock *ledger.SnapshotBlock, err error))
	Close() error
}

type SyncCache interface {
	NewWriter(from, to uint64) (io.WriteCloser, error)
	Chunks() SegmentList
	NewReader() (ReadCloser, error)
}
