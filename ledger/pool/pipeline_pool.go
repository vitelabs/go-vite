package pool

import "github.com/vitelabs/go-vite/v2/net"

func (pl *pool) AddPipeline(reader net.ChunkReader) {
	pl.pipelines = append(pl.pipelines, reader)
}
