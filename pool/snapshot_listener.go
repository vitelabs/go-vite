package pool

import (
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type snapshotPrinter struct {
	chunks chan *ledger.SnapshotChunk
	closed chan struct{}
}

func newSnapshotPrinter(closed chan struct{}) *snapshotPrinter {
	printer := &snapshotPrinter{}
	printer.chunks = make(chan *ledger.SnapshotChunk, 10000)
	printer.closed = closed
	return printer
}

func (printer *snapshotPrinter) start() {
	go printer.run()
}

func (printer *snapshotPrinter) stop() {
	close(printer.closed)
}

func (printer *snapshotPrinter) run() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-printer.closed:
			return
		case <-ticker.C:
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
			printer.print(content)
		}
	}
}

func (printer *snapshotPrinter) print(chunks []*ledger.SnapshotChunk) {
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

func (printer *snapshotPrinter) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (printer *snapshotPrinter) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	for _, v := range chunks {
		select {
		case printer.chunks <- v:
		default:
			return nil
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
