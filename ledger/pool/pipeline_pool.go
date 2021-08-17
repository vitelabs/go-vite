package pool

import "github.com/vitelabs/go-vite/net"

func (pl *pool) AddPipeline(reader net.ChunkReader) {
	pl.pipelines = append(pl.pipelines, reader)
}
