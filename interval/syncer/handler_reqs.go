package syncer

import (
	"github.com/vitelabs/go-vite/interval/common"
)

func split(buf []common.HashHeight, lim int) [][]common.HashHeight {
	var chunk []common.HashHeight
	chunks := make([][]common.HashHeight, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}
