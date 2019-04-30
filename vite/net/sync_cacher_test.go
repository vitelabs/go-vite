package net

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type mockBlocks []uint64

func (s mockBlocks) Len() int {
	return len(s)
}

func (s mockBlocks) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s mockBlocks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type mockCacheChain struct {
	height uint64
	blocks mockBlocks
	chunks chunks
	mu     sync.Mutex
	cache  *mockCache
	handle func(block *ledger.SnapshotBlock) error
}

func (m *mockCacheChain) GetSyncCache() interfaces.SyncCache {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cache == nil {
		m.cache = &mockCache{
			chunks: m.chunks,
		}
	}

	return m.cache
}

func (m *mockCacheChain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &ledger.SnapshotBlock{
		Height: m.height,
	}
}

func (m *mockCacheChain) receiveAccountBlock(block *ledger.AccountBlock, source types.BlockSource) error {
	return nil
}

func (m *mockCacheChain) receiveSnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) (err error) {
	if m.handle == nil {
		err = nil
	} else {
		err = m.handle(block)
	}
	if err != nil {
		return err
	}

	fmt.Printf("receive snapshot block %d\n", block.Height)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.blocks = append(m.blocks, block.Height)
	sort.Sort(m.blocks)

	var i int
	var height uint64
	for i, height = range m.blocks {
		if height <= m.height {
			continue
		}

		if m.height+1 == height {
			m.height = height
		} else {
			break
		}
	}

	if i > 0 {
		copy(m.blocks, m.blocks[i:])
		m.blocks = m.blocks[:len(m.blocks)-i]
	}

	return nil
}

type mockCache struct {
	chunks chunks
	mu     sync.Mutex
}

type mockCacheReader struct {
	from, to uint64
	height   uint64
}

func (m *mockCacheReader) Read() (accountBlock *ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock, err error) {
	if m.height > m.to {
		return nil, nil, io.EOF
	}

	if m.height < m.from {
		m.height = m.from
	}

	block := &ledger.SnapshotBlock{
		Height: m.height,
	}
	m.height++

	return nil, block, nil
}

func (m *mockCacheReader) Close() error {
	return nil
}

type mockCacheWriter struct {
	from, to uint64
	cache    *mockCache
}

func (m *mockCacheWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockCacheWriter) Close() error {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()

	if chunk, ok := m.cache.chunks.overlap(m.from, m.to); ok {
		m.cache.chunks = append(m.cache.chunks, [2]uint64{m.from, m.to})
		sort.Sort(m.cache.chunks)
	} else {
		panic(fmt.Sprintf("collect: chunk %d-%d with %d-%d", chunk[0], chunk[1], m.from, m.to))
	}

	return nil
}

func (m *mockCache) NewWriter(segment interfaces.Segment) (io.WriteCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	from, to := segment.Bound[0], segment.Bound[1]
	if chunk, ok := m.chunks.overlap(segment.Bound[0], segment.Bound[1]); ok {
		return &mockCacheWriter{
			from:  from,
			to:    to,
			cache: m,
		}, nil
	} else {
		panic(fmt.Sprintf("collect: chunk %d-%d with %d-%d", chunk[0], chunk[1], from, to))
	}
}

func (m *mockCache) Chunks() interfaces.SegmentList {
	m.mu.Lock()
	defer m.mu.Unlock()

	sort.Sort(m.chunks)

	segments := make(interfaces.SegmentList, len(m.chunks))
	for i, c := range m.chunks {
		segments[i] = interfaces.Segment{
			Bound: c,
		}
	}

	return segments
}

func (m *mockCache) NewReader(segment interfaces.Segment) (interfaces.ReadCloser, error) {
	return &mockCacheReader{
		from: segment.Bound[0],
		to:   segment.Bound[1],
	}, nil
}

func (m *mockCache) Delete(segment interfaces.Segment) error {
	fmt.Printf("delete chunk %d-%d\n", segment.Bound[0], segment.Bound[1])

	m.mu.Lock()
	defer m.mu.Unlock()

	for i, c := range m.chunks {
		if c == segment.Bound {
			copy(m.chunks[i:], m.chunks[i+1:])
			m.chunks = m.chunks[:len(m.chunks)-1]
		}
	}

	return nil
}

type mockSyncDownloader struct {
	listener taskListener
	chain    interface {
		GetSyncCache() interfaces.SyncCache
	}
}

func (m *mockSyncDownloader) status() DownloaderStatus {
	return DownloaderStatus{}
}

func (m *mockSyncDownloader) download(from, to uint64, must bool) bool {
	fmt.Println("download", from, to, must)
	time.Sleep(time.Second)
	w, err := m.chain.GetSyncCache().NewWriter(interfaces.Segment{
		Bound:    [2]uint64{from, to},
		Hash:     types.Hash{},
		PrevHash: types.Hash{},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create writer %d-%d: %v", from, to, err))
	}
	_ = w.Close()
	m.listener(from, to, nil)
	return true
}

func (m *mockSyncDownloader) cancel(from uint64) (end uint64) {
	fmt.Printf("cancel %d", from)
	return 0
}

func (m *mockSyncDownloader) addListener(listener taskListener) {
	m.listener = listener
}

func (m *mockSyncDownloader) start() {
	return
}

func (m *mockSyncDownloader) stop() {
	return
}

func TestSyncCache_read(t *testing.T) {
	const end = 50
	chain := &mockCacheChain{
		height: 0,
		chunks: [][2]uint64{
			{2, 9},
			{10, 20},
			{30, 40},
			{45, end},
		},
	}

	reader := newCacheReader(chain, mockVerifier{}, &mockSyncDownloader{
		chain: chain,
	})

	reader.start()

	pending := make(chan struct{})
	go func() {
		for {
			if chain.GetLatestSnapshotBlock().Height == end {
				pending <- struct{}{}
				break
			} else {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	<-pending

	reader.stop()

	if chain.GetLatestSnapshotBlock().Height != end {
		t.Fail()
	}
}

func TestSyncCache_read_error(t *testing.T) {
	const end = 50
	chain := &mockCacheChain{
		height: 10,
		chunks: [][2]uint64{
			{1, 9},
			{10, 20},
			{30, 40},
			{45, end},
		},
		handle: func(block *ledger.SnapshotBlock) error {
			if rand.Intn(10) > 7 {
				return fmt.Errorf("mock handle snapshotblock %d error", block.Height)
			}
			return nil
		},
	}

	reader := newCacheReader(chain, mockVerifier{}, &mockSyncDownloader{
		chain: chain,
	})

	reader.start()

	pending := make(chan struct{})
	go func() {
		for {
			if chain.GetLatestSnapshotBlock().Height == end {
				pending <- struct{}{}
				break
			} else {
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	<-pending

	reader.stop()
}

func TestChunkRead(t *testing.T) {
	// success
	const from uint64 = 101
	const to uint64 = 200
	var prevHash, hash, endHash types.Hash
	chunk := newChunk(prevHash, from-1, endHash, to, types.RemoteSync)

	for i := from; i < to+1; i++ {
		_, _ = crand.Read(hash[:])

		err := chunk.addSnapshotBlock(&ledger.SnapshotBlock{
			Hash:     hash,
			PrevHash: prevHash,
			Height:   i,
		})

		if err != nil {
			t.Errorf("error should be nil")
		}

		prevHash = hash
	}
}

func TestChunkRead2(t *testing.T) {
	const from uint64 = 101
	const to uint64 = 200
	var prevHash, hash, endHash types.Hash
	chunk := newChunk(prevHash, from-1, endHash, to, types.RemoteSync)

	_, _ = crand.Read(hash[:])

	err := chunk.addSnapshotBlock(&ledger.SnapshotBlock{
		Hash:     hash,
		PrevHash: prevHash,
		Height:   from + 1,
	})

	if err == nil {
		t.Errorf("error should not be nil")
	}

	fmt.Println(err)
}
