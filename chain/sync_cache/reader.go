package sync_cache

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/golang/snappy"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/ledger"
)

type Reader struct {
	cache        *syncCache
	file         *os.File
	readBuffer   []byte
	decodeBuffer []byte
	item         *cacheItem
}

func (reader *Reader) Size() int64 {
	return reader.item.size
}

func NewReader(cache *syncCache, item *cacheItem) (*Reader, error) {
	if !item.done {
		return nil, fmt.Errorf("failed to open cache %d-%d %s-%s: not write done", item.From, item.To, item.PrevHash, item.Hash)
	}

	file, err := os.OpenFile(item.filename, os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache %d-%d %s-%s: %v", item.From, item.To, item.PrevHash, item.Hash, err)
	}

	r := &Reader{
		cache:      cache,
		file:       file,
		readBuffer: make([]byte, 8*1024),
		item:       item,
	}

	return r, nil
}

// Verified is not thread-safe
func (reader *Reader) Verified() bool {
	return reader.item.verified
}

// Verify is not thread-safe
func (reader *Reader) Verify() {
	if reader.Verified() {
		return
	}

	reader.item.verified = true
	_ = reader.cache.updateIndex(reader.item)
}

func (reader *Reader) Read() (ab *ledger.AccountBlock, sb *ledger.SnapshotBlock, err error) {
	fd := reader.file

	// 4 bytes for payload length size
	buf := reader.readBuffer[:4]
	if _, err = fd.Read(buf); err != nil {
		return
	}

	size := binary.BigEndian.Uint32(buf)
	if cap(reader.readBuffer) < int(size) {
		reader.readBuffer = make([]byte, size)
	}

	if size == 0 {
		return nil, nil, errors.New("0 size")
	}

	buf = reader.readBuffer[:size]
	if _, err = fd.Read(buf); err != nil {
		return
	}

	if len(buf) != int(size) {
		return nil, nil, errors.New("error size length")
	}

	// first payload byte is code
	// rest is data
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
	return reader.file.Close()
}
