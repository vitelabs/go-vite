package main

import (
	"fmt"
	"io"
	"time"

	"github.com/vitelabs/go-vite/pool"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/chain"
)

const chunkSize = 100

type syncer struct {
	chain    chain.Chain
	pool     pool.BlockPool
	from, to uint64
}

func newSyncer(dirname string) *syncer {
	c, err := newChain(dirname)
	if err != nil {
		panic(err)
	}

	fmt.Println(c.GetLatestSnapshotBlock())

	p, err := pool.NewPool(c)
	if err != nil {
		panic(err)
	}

	return &syncer{
		chain: c,
		pool:  p,
	}
}

func (s *syncer) height() uint64 {
	return s.chain.GetLatestSnapshotBlock().Height
}

func (s *syncer) request(from, to uint64, w io.Writer) error {
	r, err := s.chain.GetLedgerReaderByHeight(from, to)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	return err
}

func (s *syncer) readCacheLoop() {
Loop:
	for {
		if s.height() >= s.to {
			return
		}

		cache := s.chain.GetSyncCache()
		cs := cache.Chunks()

		if len(cs) == 0 {
			time.Sleep(3 * time.Second)
			continue
		}

		var err error
		var reader interfaces.ReadCloser

		c := cs[0]
		if c.Bound[1] > s.height() {
			reader, err = cache.NewReader(c)
			if err != nil {
				goto Loop
			}

			var prev *ledger.SnapshotBlock
			for {
				var ab *ledger.AccountBlock
				var sb *ledger.SnapshotBlock
				ab, sb, err = reader.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					break
				} else if ab != nil {
					//err = s.pool.AddAccountBlock(ab)
					//if err != nil {
					//}
					//break
				} else if sb != nil {
					if prev == nil {
						if sb.Height != c[0] {
							err = fmt.Errorf("wrong start: %d %d", sb.Height, c[0])
							panic(err)
						}
						prev = sb
					} else {
						if prev.Hash != sb.PrevHash {
							err = fmt.Errorf("not continuous: %s/%d %s/%s/%d", prev.Hash, prev.Height, sb.PrevHash, sb.Hash, sb.Height)
							panic(err)
						} else {
							prev = sb
						}
					}
					//err = s.receiveSnapshotBlock(sb)
					//if err != nil {
					//}
					fmt.Println(sb.Height, sb.Hash, sb.PrevHash)
					//break
				}
			}

			if prev.Height != c.Bound[1] {
				err = fmt.Errorf("wrong end: %d %d", prev.Height, c.Bound[1])
				panic(err)
			}

			_ = reader.Close()
		}
	}
}
