package sync_cache

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/golang/snappy"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

func (cache *syncCache) NewReader(segment interfaces.Segment) (interfaces.ReadCloser, error) {
	cache.segMu.RLock()
	defer cache.segMu.RUnlock()

	if !cache.CheckExisted(segment) {
		return nil, errors.New(fmt.Sprintf("file is not existed, %s/%d %s/%d", segment.PrevHash, segment.Bound[0], segment.Hash, segment.Bound[1]))
	}

	return NewReader(cache, segment)
}

func (cache *syncCache) CheckExisted(segment interfaces.Segment) bool {
	chunks := cache.Chunks()
	if chunks.Len() <= 0 {
		return false
	}

	for _, chunk := range chunks {
		if segment.Hash == chunk.Hash &&
			segment.PrevHash == chunk.PrevHash {
			return true
		}
	}
	return false
}

type Reader struct {
	cache        *syncCache
	file         *os.File
	size         int64
	offset       int64
	readBuffer   []byte
	decodeBuffer []byte
}

func NewReader(cache *syncCache, seg interfaces.Segment) (*Reader, error) {
	fileName := cache.toAbsoluteFileName(seg)
	file, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	r := &Reader{
		cache:      cache,
		file:       file,
		offset:     0,
		readBuffer: make([]byte, 8*1024),
	}

	fst, err := os.Stat(fileName)
	if err == nil {
		r.size = fst.Size()
	}

	return r, nil
}

func (reader *Reader) Size() int64 {
	return reader.size
}

func (reader *Reader) Read() (ab *ledger.AccountBlock, sb *ledger.SnapshotBlock, err error) {
	fd := reader.file

	buf := reader.readBuffer[:4]
	if _, err = fd.Read(buf); err != nil {
		return
	}

	size := binary.BigEndian.Uint32(buf)
	if cap(reader.readBuffer) < int(size) {
		reader.readBuffer = make([]byte, size)
	}

	buf = reader.readBuffer[:size]
	if _, err = fd.Read(buf); err != nil {
		return
	}
	code := buf[0]

	decodeLen, err := snappy.DecodedLen(buf[1:])
	if err != nil {
		return
	}
	if cap(reader.decodeBuffer) < decodeLen {
		reader.decodeBuffer = make([]byte, decodeLen)
	}

	sBuf, err := snappy.Decode(reader.decodeBuffer, buf[1:])
	if err != nil {
		return
	}

	switch code {
	case chain_block.BlockTypeAccountBlock:
		ab = &ledger.AccountBlock{}
		if err = ab.Deserialize(sBuf); err != nil {
			return
		}
		return
	case chain_block.BlockTypeSnapshotBlock:
		sb = &ledger.SnapshotBlock{}
		if err = sb.Deserialize(sBuf); err != nil {
			return
		}
		return
	}

	return nil, nil, io.EOF
}

func (reader *Reader) Close() error {
	return reader.close()
}

func (reader *Reader) close() error {
	return reader.file.Close()
}

func (cache *syncCache) deleteSeg(segToDelete interfaces.Segment) {
	cache.segMu.Lock()
	defer cache.segMu.Unlock()

	for index, seg := range cache.segments {
		if seg.Hash == segToDelete.Hash &&
			seg.PrevHash == segToDelete.PrevHash {

			cache.segments = append(cache.segments[:index], cache.segments[index+1:]...)
			return
		}
	}

}
