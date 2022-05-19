package pipeline

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
	chain_block "github.com/vitelabs/go-vite/v2/ledger/chain/block"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/net"
)

var (
	log = log15.New("module", "ledger/pipeline")
)

type blocks_pipeline struct {
	net.ChunkReader

	fromDir    string
	fromBlocks *blocks

	// --------------
	chunkCh  chan *core.SnapshotChunk
	curChunk *net.Chunk
}

func newBlocksPipeline(fromDir string, fileSize int64) (*blocks_pipeline, error) {
	p := &blocks_pipeline{}
	p.fromDir = fromDir
	bs, err := newBlocks(fromDir, fileSize)
	if err != nil {
		return nil, err
	}
	p.fromBlocks = bs
	return p, nil
}

func newBlocksPipelineWithRun(fromDir string, height uint64, fileSize int64) (*blocks_pipeline, error) {
	p, err := newBlocksPipeline(fromDir, fileSize)
	if err != nil {
		log.Error("new blocks pipeline fail.", "err", err)
		return nil, err
	}
	p.chunkCh = make(chan *core.SnapshotChunk, 1000)

	if height > 100 {
		height = height - 100
	}

	location, err := p.fromBlocks.location(height)
	if err != nil {
		log.Error("new blocks pipeline fail. read location fail", "err", err)
		return nil, err
	}

	go func() {
		defer close(p.chunkCh)
		for {
			chunk, next, err := p.fromBlocks.blockDb.ReadChunk(*location)
			if err != nil {
				log.Error("read chunk fail.", "err", err, "location", location, "height", height)
				return
			}
			log.Debug(fmt.Sprintf("pipeline chunk to %d", chunk.SnapshotBlock.Height))
			p.chunkCh <- chunk
			location = next
		}
	}()
	return p, nil
}

func NewBlocksPipeline(fromDir string, height uint64) (*blocks_pipeline, error) {
	return newBlocksPipelineWithRun(fromDir, height, chain_block.FixFileSize)
}

func (p *blocks_pipeline) Peek() *net.Chunk {
	if p.curChunk != nil {
		return p.curChunk
	} else {
		chunk := p.read()
		p.curChunk = chunk
		return chunk
	}
}

func (p *blocks_pipeline) read() *net.Chunk {
	i := 0
	var chunks []core.SnapshotChunk
	for {
		i++
		if i > 1000 {
			return net.NewChunk(chunks, types.Local)
		}
		select {
		case chunk, ok := <-p.chunkCh:
			if !ok && chunk == nil {
				return net.NewChunk(chunks, types.Local)
			}
			chunks = append(chunks, *chunk)
		default:
			return net.NewChunk(chunks, types.Local)
		}
	}
}

func (p *blocks_pipeline) Pop(hash types.Hash) {
	if p.curChunk != nil && p.curChunk.SnapshotRange[1].Hash == hash {
		p.curChunk = nil
	}
}
