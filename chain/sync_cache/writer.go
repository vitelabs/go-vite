package sync_cache

import (
	"os"
)

type writer struct {
	cache *syncCache
	item  *cacheItem
	fd    *os.File
}

func (w *writer) Write(p []byte) (n int, err error) {
	return w.fd.Write(p)
}

func (w *writer) Close() (err error) {
	// close file
	err = w.fd.Close()
	if err != nil {
		w.cache.deleteItem(w.item)
		return
	}

	w.item.done = true

	// add item to index db
	err = w.cache.updateIndex(w.item)

	return
}
