package pool

import (
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/net"
)

type snapshotPrinter struct {
	chunks    chan *ledger.SnapshotChunk
	snapshots chan *ledger.SnapshotBlock
	closed    chan struct{}
	sync      syncer
}

func newSnapshotPrinter(closed chan struct{}, sync syncer) *snapshotPrinter {
	printer := &snapshotPrinter{}
	printer.chunks = make(chan *ledger.SnapshotChunk, 10000)
	printer.snapshots = make(chan *ledger.SnapshotBlock, 100)
	printer.closed = closed
	printer.sync = sync
	return printer
}

func (printer *snapshotPrinter) start() {
	go printer.run()
}

func (printer *snapshotPrinter) stop() {
	close(printer.closed)
}

func (printer *snapshotPrinter) run() {
	statsTicker := time.NewTicker(time.Second * 10)
	defer statsTicker.Stop()

	for {
		select {
		case <-printer.closed:
			return
		case sblock := <-printer.snapshots:
			// every 10s
			printer.printSnapshot(sblock)
		case <-statsTicker.C:
			// every 10s
			var content []*ledger.SnapshotChunk
		Read:
			for {
				select {
				case c := <-printer.chunks:
					content = append(content, c)
				case <-printer.closed:
					return
				default:
					break Read
				}
			}
			printer.printStats(content)
		}
	}
}

func (printer *snapshotPrinter) printStats(chunks []*ledger.SnapshotChunk) {
	if len(chunks) == 0 {
		return
	}
	chunk := chunks[len(chunks)-1]
	block := chunk.SnapshotBlock
	if block == nil {
		return
	}
	fmt.Printf("[Snapshot Stats] Height:%d, Hash:%s, Timestamp:%s, Producer:%s, Time:%s\n", block.Height, block.Hash, block.Timestamp, block.Producer(), time.Now())
}

func (printer *snapshotPrinter) printSnapshot(sBlock *ledger.SnapshotBlock) {
	if sBlock == nil {
		return
	}
	fmt.Printf("[Snapshot Insert] Height:%d, Hash:%s, Timestamp:%s, Producer:%s, Time:%s\n", sBlock.Height, sBlock.Hash, sBlock.Timestamp, sBlock.Producer(), time.Now())
}

func (printer *snapshotPrinter) PrepareInsertAccountBlocks(blocks []*interfaces.VmAccountBlock) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) InsertAccountBlocks(blocks []*interfaces.VmAccountBlock) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	if printer.sync.SyncState() == net.SyncDone {
		for _, v := range chunks {
			select {
			case printer.snapshots <- v.SnapshotBlock:
			default:
				return nil
			}
		}
	} else {
		for _, v := range chunks {
			select {
			case printer.chunks <- v:
			default:
				return nil
			}
		}
	}
	return nil
}

func (printer *snapshotPrinter) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}
