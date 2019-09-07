package interfaces

import (
	"io"
	"math/big"
	"strconv"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type StorageIterator interface {
	Last() bool
	Prev() bool
	Seek(key []byte) bool

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
	Seg() Segment
	Size() int
	io.ReadCloser
}

type Segment struct {
	From, To uint64
	Hash     types.Hash
	PrevHash types.Hash
	Points   []*ledger.HashHeight
}

func (seg Segment) String() string {
	return strconv.FormatUint(seg.From, 10) + "-" + strconv.FormatUint(seg.To, 10) + " " + seg.PrevHash.String() + "-" + seg.Hash.String()
}

func (seg Segment) Equal(seg2 Segment) bool {
	return seg.From == seg2.From && seg.To == seg2.To && seg.PrevHash == seg2.PrevHash && seg.Hash == seg2.Hash
}

type SegmentList []Segment

func (list SegmentList) Len() int { return len(list) }
func (list SegmentList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
func (list SegmentList) Less(i, j int) bool {
	return list[i].From < list[j].To
}

type ChunkReader interface {
	// Read a block, return io.EOF if reach end, the block maybe a accountBlock or a snapshotBlock
	Read() (accountBlock *ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock, err error)
	// Close the stream
	Close() error
	Size() int64
	Verified() bool
	Verify()
}

type SyncCache interface {
	NewWriter(segment Segment, size int64) (io.WriteCloser, error)
	Chunks() SegmentList
	NewReader(segment Segment) (ChunkReader, error)
	Delete(seg Segment) error
	Close() error
}

type DBStatus struct {
	Name   string
	Count  uint64
	Size   uint64
	Status string
}
