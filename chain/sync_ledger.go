package chain

import "github.com/vitelabs/go-vite/interfaces"

func (c *chain) GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error) {
	return nil, nil
}

func (c *chain) GetSyncCache() interfaces.SyncCache {
	return c.syncCache
}
