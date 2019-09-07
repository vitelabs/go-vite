package sync_cache

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	leveldb "github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/log15"
)

const filePrefix = "f_"
const indexDBName = "db"

type syncCache struct {
	dirName string

	indexDB *leveldb.DB

	caches cacheItems
	mu     sync.RWMutex

	log log15.Logger
}

func NewSyncCache(dirName string) (cache *syncCache, err error) {
	cache = &syncCache{
		log:     log15.New("module", "sync_cache"),
		dirName: dirName,
	}

	err = cache.loadCaches()
	if err != nil {
		return nil, fmt.Errorf("failed to load caches: %v", err)
	}

	return cache, nil
}

func (cache *syncCache) Close() (err error) {
	err = cache.indexDB.Close()
	return
}

func (cache *syncCache) open() (err error) {
	st, err := os.Stat(cache.dirName)
	if err != nil || !st.IsDir() {
		if os.IsNotExist(err) {
			err = os.Mkdir(cache.dirName, 0744)
			if err != nil {
				return fmt.Errorf("failed to create cache dir %s: %v", cache.dirName, err)
			}
		} else {
			if !st.IsDir() {
				err = errors.New("not dir")
			}

			cache.log.Error(fmt.Sprintf("failed to open cache dir %s: %v", cache.dirName, err))

			_ = os.RemoveAll(cache.dirName)
			err = os.Mkdir(cache.dirName, 0744)
			if err != nil {
				return fmt.Errorf("failed to create cache dir %s: %v", cache.dirName, err)
			}
		}
	}

	return
}

func (cache *syncCache) loadCaches() (err error) {
	err = cache.open()
	if err != nil {
		return
	}

	err = cache.readIndex()
	if err != nil {
		return
	}

	keepFiles := make(map[string]struct{}) // filename

	var broken bool
	for i, item := range cache.caches {
		if _, err = os.Stat(item.filename); err != nil || false == item.done {
			broken = true
			cache.log.Warn(fmt.Sprintf("failed to read cache file %s info: %v", item.filename, err))
			cache.caches[i] = nil
			cache.cleanItem(item)
		} else {
			keepFiles[item.filename] = struct{}{}
		}
	}

	if broken {
		var j int
		for _, item := range cache.caches {
			if item != nil {
				cache.caches[j] = item
				j++
			}
		}

		cache.caches = cache.caches[:j]
	}

	files, err := ioutil.ReadDir(cache.dirName)
	if err == nil {
		for _, file := range files {
			// a file not a dir
			if false == file.IsDir() {
				// remove useless file, like a unfinished writer file
				if _, ok := keepFiles[file.Name()]; !ok {
					_ = os.Remove(path.Join(cache.dirName, file.Name()))
				}
			}
		}
	}

	return
}

// should open index file first
func (cache *syncCache) readIndex() (err error) {
	indexPath := path.Join(cache.dirName, indexDBName)
	st, err := os.Stat(indexPath)
	// broken index or missing index or indexPath is not a dir
	if err != nil || !st.IsDir() {
		err = os.RemoveAll(cache.dirName)
		if err != nil {
			return fmt.Errorf("please delete sync_cache dir")
		}
		err = cache.open()
		if err != nil {
			return err
		}
	}

	cache.indexDB, err = leveldb.OpenFile(indexPath, nil)
	if err != nil {
		err = fmt.Errorf("failed to open cache index db: %v", err)
		return
	}

	itr := cache.indexDB.NewIterator(util.BytesPrefix(dbItemPrefix), nil)
	defer itr.Release()

	for itr.Next() {
		item := &cacheItem{}
		err = item.DeSerialize(itr.Value())
		if err != nil {
			_ = cache.indexDB.Delete(itr.Key(), nil)
			continue
		}

		cache.caches = append(cache.caches, item)
	}

	sort.Sort(cache.caches)

	return
}

func (cache *syncCache) updateIndex(item *cacheItem) (err error) {
	data, err := item.Serialize()
	if err != nil {
		cache.log.Warn(fmt.Sprintf("failed to serialize item: %v", err))
		return
	}

	err = cache.indexDB.Put(item.dbKey(), data, nil)
	if err != nil {
		cache.log.Warn(fmt.Sprintf("failed to store item: %v", err))
	}
	return
}

func (cache *syncCache) NewReader(segment interfaces.Segment) (interfaces.ChunkReader, error) {
	item, ok := cache.findSeg(segment)

	if ok {
		return NewReader(cache, item)
	}

	return nil, fmt.Errorf("failed to find cache: %d-%d %s-%s", segment.From, segment.To, segment.PrevHash, segment.Hash)
}

// NewWriter will add a temp chunk to chunk list, and create a file bind to the temp chunk, return the file to write.
// the temp chunk will not add to index db until writer is done.
func (cache *syncCache) NewWriter(segment interfaces.Segment, size int64) (w io.WriteCloser, err error) {
	cache.mu.Lock()

	index, ok := cache.checkOverlap(segment)
	if ok {
		cache.mu.Unlock()
		return nil, fmt.Errorf("failed to cache %d-%d: overlapped", segment.From, segment.To)
	}

	item := &cacheItem{
		Segment:  segment,
		done:     false,
		verified: false,
		filename: "",
		size:     size,
	}

	cache.caches = append(cache.caches, nil)
	copy(cache.caches[index+1:], cache.caches[index:])
	cache.caches[index] = item

	cache.mu.Unlock()

	file, err := cache.createNewFile(item)
	if err != nil {
		cache.deleteItem(item)
		return
	}

	item.filename = file.Name()

	w = &writer{
		cache: cache,
		item:  item,
		fd:    file,
	}

	return
}

func (cache *syncCache) createNewFile(item *cacheItem) (file *os.File, err error) {
	filename := cache.toCacheFileName(item.Segment)
	file, err = os.Create(filename)

	if err != nil {
		err = fmt.Errorf("failed to create cache file %s: %v", filename, err)
	}

	return
}

func (cache *syncCache) checkOverlap(segment interfaces.Segment) (index int, overlapped bool) {
	n := sort.Search(len(cache.caches), func(i int) bool {
		return cache.caches[i].From > segment.To
	})

	if n == 0 || cache.caches[n-1].To < segment.From {
		return n, false
	}

	return -1, true
}

func (cache *syncCache) Delete(seg interfaces.Segment) error {
	item, ok := cache.findSeg(seg)

	if ok {
		cache.deleteItem(item)

		return nil
	} else {
		return fmt.Errorf("failed to find segment: %d-%d %s-%s", seg.From, seg.To, seg.PrevHash, seg.Hash)
	}
}

func (cache *syncCache) findSeg(seg interfaces.Segment) (item *cacheItem, ok bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	for _, item = range cache.caches {
		if item.Equal(seg) {
			ok = true
			break
		}
	}

	return
}

func (cache *syncCache) deleteItem(toDelete *cacheItem) {
	var find bool

	cache.mu.Lock()
	for index, item := range cache.caches {
		if item == toDelete {
			find = true
			cache.caches = append(cache.caches[:index], cache.caches[index+1:]...)
			break
		}
	}
	cache.mu.Unlock()

	if find {
		cache.cleanItem(toDelete)
	}
}

func (cache *syncCache) cleanItem(item *cacheItem) {
	var err error
	err = cache.indexDB.Delete(item.dbKey(), nil)
	if err != nil {
		cache.log.Warn(fmt.Sprintf("failed to delete item %d-%d from db: %v", item.From, item.To, err))
	}

	if item.filename != "" {
		err = os.Remove(item.filename)
		if err != nil {
			cache.log.Warn(fmt.Sprintf("failed to delete item file %s: %v", item.filename, err))
		}
	}
}

func (cache *syncCache) Chunks() interfaces.SegmentList {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	cs := make(interfaces.SegmentList, 0, len(cache.caches))
	for _, item := range cache.caches {
		if item.done {
			cs = append(cs, item.Segment)
		}
	}

	return cs
}

func (cache *syncCache) toCacheFileName(segment interfaces.Segment) string {
	return path.Join(cache.dirName, toFilename(filePrefix, segment))
}

func toFilename(prefix string, segment interfaces.Segment) string {
	return prefix + strconv.FormatUint(segment.From, 10) + "_" + strconv.FormatUint(segment.To, 10) + "_" + strconv.FormatInt(time.Now().Unix(), 10)
}
