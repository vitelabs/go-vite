package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/interfaces"
	"io"
)

func (c *chain) GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error) {
	if startHeight <= 0 {
		return nil, errors.New("startHeight is 0")
	}
	if startHeight < endHeight {
		return nil, errors.New(fmt.Sprintf("startHeight < endHeight, startHeight is %d, endHeight is %d", startHeight, endHeight))
	}
	latestSnapshotBlock := c.GetLatestSnapshotBlock()
	if endHeight > latestSnapshotBlock.Height {
		return nil, errors.New(fmt.Sprintf("endHeight is too big, endHeight is %d, latest snapshot height is %d", endHeight, latestSnapshotBlock.Height))
	}

	return newLedgerReader(c, startHeight, endHeight)

}

func (c *chain) GetSyncCache() interfaces.SyncCache {
	return c.syncCache
}

type ledgerReader struct {
	chain *chain

	from uint64
	to   uint64

	fromLocation *chain_file_manager.Location
	toLocation   *chain_file_manager.Location

	currentLocation *chain_file_manager.Location
}

func newLedgerReader(chain *chain, from, to uint64) (interfaces.LedgerReader, error) {
	fromLocation, err := chain.indexDB.GetSnapshotBlockLocation(from)
	if err != nil {
		return nil, err
	}
	if fromLocation == nil {
		return nil, errors.New(fmt.Sprintf("snapshot %d is not existed", from))
	}

	toLocation, err := chain.indexDB.GetSnapshotBlockLocation(to)
	if err != nil {
		return nil, err
	}
	if toLocation == nil {
		return nil, errors.New(fmt.Sprintf("snapshot %d is not existed", to))
	}
	return &ledgerReader{
		chain:        chain,
		from:         from,
		to:           to,
		fromLocation: fromLocation,
		toLocation:   toLocation,
	}, nil
}
func (reader *ledgerReader) Bound() (from, to uint64) {
	return reader.from, reader.to
}

func (reader *ledgerReader) Size() int {
	return reader.size(reader.fromLocation, reader.toLocation)
}

func (reader *ledgerReader) Read(p []byte) (n int, err error) {
	readN := reader.size(reader.currentLocation, reader.toLocation)
	isEnd := false
	if readN <= len(p) {
		isEnd = true
	} else {
		readN = len(p)
	}
	currentLocation, n, err := reader.chain.blockDB.ReadRaw(reader.currentLocation, p[:readN])
	if err != nil {
		return n, err
	}
	reader.currentLocation = currentLocation

	if isEnd {
		err = io.EOF
	}

	return n, err
}

func (reader *ledgerReader) Close() error {
	reader.currentLocation = reader.toLocation
	return nil
}

func (reader *ledgerReader) size(frontLocation, backLocation *chain_file_manager.Location) int {
	return int(backLocation.FileId-frontLocation.FileId)*int(reader.chain.blockDB.FileSize()) +
		int(backLocation.Offset-frontLocation.Offset)
}
