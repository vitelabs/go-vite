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
		return nil, errors.New(fmt.Sprintf("file is not existed, from is %d, to is %d", from, to))
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
	cache            *syncCache
	file             *os.File
	offset           int64
	snappyReadBuffer []byte
}

func NewReader(cache *syncCache, seg interfaces.Segment) (*Reader, error) {
	fileName := cache.toAbsoluteFileName(seg)
	file, oErr := os.OpenFile(fileName, os.O_RDWR, 0666)
	if oErr != nil {
		if os.IsNotExist(oErr) {
			return nil, nil
		}
		return nil, oErr
	}

	return &Reader{
		cache:            cache,
		file:             file,
		offset:           0,
		snappyReadBuffer: make([]byte, 0, 8*1024),
	}, nil
}

func (reader *Reader) Read() (accountBlock *ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock, returnErr error) {
	defer func() {
		if returnErr != nil {
			reader.close()
		}
	}()
	fd := reader.file

	bufSizeBytes := make([]byte, 4)
	if _, err := fd.Read(bufSizeBytes); err != nil {
		return nil, nil, err
	}
	bufSize := binary.BigEndian.Uint32(bufSizeBytes)

	buf := make([]byte, bufSize)
	if _, err := fd.Read(buf); err != nil {
		return nil, nil, err
	}

	sBuf, err := snappy.Decode(reader.snappyReadBuffer, buf[1:])
	if err != nil {
		return nil, nil, err
	}

	switch buf[0] {
	case chain_block.BlockTypeAccountBlock:
		ab := &ledger.AccountBlock{}
		if err := ab.Deserialize(sBuf); err != nil {
			return nil, nil, err
		}
		return ab, nil, nil
	case chain_block.BlockTypeSnapshotBlock:
		sb := &ledger.SnapshotBlock{}
		if err := sb.Deserialize(sBuf); err != nil {
			return nil, nil, err
		}
		return nil, sb, nil
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
