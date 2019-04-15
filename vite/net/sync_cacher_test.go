package net

import (
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/interfaces"

	"github.com/vitelabs/go-vite/ledger"
)

type mockCacheChain struct {
	height uint64
	chunks [][2]uint64
}

type mockCache struct {
	chunks [][2]uint64
	mu     sync.Mutex
}

type mockCacheReader struct {
}

func (m *mockCacheReader) Read() (accountBlock *ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock, err error) {
	return nil, nil, io.EOF
}

func (m *mockCacheReader) Close() error {
	return nil
}

type mockCacheWriter struct {
}

func (m *mockCacheWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockCacheWriter) Close() error {
	return nil
}

func (m *mockCache) NewWriter(from, to uint64) (io.WriteCloser, error) {
	return &mockCacheWriter{}, nil
}

func (m *mockCache) Chunks() interfaces.SegmentList {
	m.mu.Lock()
	defer m.mu.Unlock()

	segs := make(interfaces.SegmentList, len(m.chunks))
	for i, c := range m.chunks {
		segs[i] = c
	}

	return segs
}

func (m *mockCache) NewReader(from, to uint64) (interfaces.ReadCloser, error) {
	return &mockCacheReader{}, nil
}

func (m *mockCache) Delete(seg interfaces.Segment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.chunks {
		if c == seg {
			copy(m.chunks[i:], m.chunks[i+1:])
			m.chunks = m.chunks[:len(c)-1]
		}
	}

	return nil
}

func (m *mockCacheChain) GetSyncCache() interfaces.SyncCache {
	return &mockCache{
		chunks: m.chunks,
	}
}

func (m *mockCacheChain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	m.height++
	return &ledger.SnapshotBlock{
		Height: m.height,
	}
}

type mockReceiver struct {
}

func (m *mockReceiver) receiveAccountBlock(block *ledger.AccountBlock, source types.BlockSource) error {
	return nil
}

func (m *mockReceiver) receiveSnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) error {
	return nil
}

type mockSyncDownloader struct {
}

func (m *mockSyncDownloader) download(from, to uint64, must bool) bool {
	fmt.Println("download", from, to, must)
	return true
}

func (m *mockSyncDownloader) cancel(from, to uint64) {

}

func (m *mockSyncDownloader) addListener(listener taskListener) {

}

func (m *mockSyncDownloader) start() {
	return
}

func (m *mockSyncDownloader) stop() {
	return
}

func TestSyncCache(t *testing.T) {
	reader := newCacheReader(&mockCacheChain{
		height: 10,
		chunks: [][2]uint64{
			{1, 9},
			{10, 20},
			{30, 40},
			{45, 50},
		},
	}, &mockReceiver{}, &mockSyncDownloader{})

	reader.start()

	time.Sleep(time.Second)
}
