package interfaces

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"io"
	"math/big"
)

type StorageIterator interface {
	Last() bool
	Prev() bool
	Next() bool

	Key() []byte
	Value() []byte
	Error() error
	Release()
}

type StateSnapshot interface {
	// ====== balance ======
	GetBalance(tokenId *types.TokenTypeId) (*big.Int, error)

	// ====== Storage ======
	GetValue([]byte) ([]byte, error)

	NewStorageIterator(prefix []byte) StorageIterator

	Release()
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
}

type Store interface {
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
}

type LedgerReader interface {
	Bound() (from, to uint64)
	Size() int
	Stream() io.ReadCloser
}

type ReadCloser interface {
	// Read a block, return io.EOF if reach end, the block maybe a accountBlock or a snapshotBlock
	Read() (accountBlock *ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock, err error)
	// Close the stream
	Close() error
}

type Segment [2]uint64
type SegmentList []Segment

func (list SegmentList) Len() int { return len(list) }
func (list SegmentList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
func (list SegmentList) Less(i, j int) bool {
	return list[i][0] < list[j][1]
}

type SyncCache interface {
	NewWriter(from, to uint64) (io.WriteCloser, error)
	Chunks() SegmentList
	NewReader(from, to uint64) (ReadCloser, error)
}
