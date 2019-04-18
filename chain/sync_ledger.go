package chain

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/interfaces"
)

func (c *chain) GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error) {
	if startHeight < 2 {
		return nil, errors.New(fmt.Sprintf("startHeight is %d", startHeight))
	}
	if startHeight > endHeight {
		return nil, errors.New(fmt.Sprintf("startHeight > endHeight, startHeight is %d, endHeight is %d", startHeight, endHeight))
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
	tmpFromLocation, err := chain.indexDB.GetSnapshotBlockLocation(from - 1)
	if err != nil {
		return nil, err
	}
	if tmpFromLocation == nil {
		return nil, errors.New(fmt.Sprintf("from location %d is not existed", tmpFromLocation))
	}

	fromLocation, err := chain.blockDB.GetNextLocation(tmpFromLocation)
	if err != nil {
		return nil, err
	}
	if fromLocation == nil {
		return nil, errors.New(fmt.Sprintf("snapshot %d is not existed", from))
	}

	tmpToLocation, err := chain.indexDB.GetSnapshotBlockLocation(to)
	if err != nil {
		return nil, err
	}
	if tmpToLocation == nil {
		return nil, errors.New(fmt.Sprintf("snapshot %d is not existed", to))
	}

	toLocation, err := chain.blockDB.GetNextLocation(tmpToLocation)
	if err != nil {
		return nil, err
	}
	if toLocation == nil {
		return nil, errors.New(fmt.Sprintf("next location %d is not existed", toLocation))
	}

	return &ledgerReader{
		chain:           chain,
		from:            from,
		to:              to,
		fromLocation:    tmpFromLocation,
		currentLocation: tmpFromLocation,
		toLocation:      toLocation,
	}, nil
}
func (reader *ledgerReader) Bound() (from, to uint64) {
	return reader.from, reader.to
}

func (reader *ledgerReader) Size() int {
	return int(reader.fromLocation.Distance(reader.chain.blockDB.FileSize(), reader.toLocation))
}

func (reader *ledgerReader) Read(p []byte) (n int, err error) {
	readN := int(reader.currentLocation.Distance(reader.chain.blockDB.FileSize(), reader.toLocation))
	isEnd := false
	if readN <= len(p) {
		isEnd = true
	} else {
		readN = len(p)
	}
	currentLocation, n, err := reader.chain.blockDB.ReadRaw(reader.currentLocation, p[:readN])

	reader.currentLocation = currentLocation
	if err != nil {
		return n, err
	}

	if isEnd {
		err = io.EOF
	}

	return n, err
}

func (reader *ledgerReader) Close() error {
	reader.currentLocation = reader.toLocation
	return nil
}
