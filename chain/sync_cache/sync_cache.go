package sync_cache

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type syncCache struct {
	log     log15.Logger
	dirName string
	dirFd   *os.File

	maxFileSize int64

	segments interfaces.SegmentList
	segMu    sync.RWMutex
}

func NewSyncCache(dataDir string) (interfaces.SyncCache, error) {
	dirName := path.Join(dataDir, "sync_cache")

	var err error
	cache := &syncCache{
		log:         log15.New("module", "sync_cache"),
		dirName:     dirName,
		maxFileSize: 20 * 1024 * 1024,
	}

	cache.dirFd, err = cache.newDirFd(dirName)

	if err != nil {
		sErr := errors.New(fmt.Sprintf("cache.newDirFd failed, error is %s, dirName is %s", err, dirName))
		cache.log.Error(sErr.Error(), "method", "NewSyncCache")
		return nil, sErr
	}

	cache.segments, err = cache.loadAllSegments()
	if err != nil {
		sErr := errors.New(fmt.Sprintf("cache.loadAllSegments failed, error is %s", err))
		cache.log.Error(sErr.Error(), "method", "NewSyncCache")
		return nil, sErr
	}

	return cache, nil
}

func (cache *syncCache) AddSeg(from, to uint64) {
	cache.segMu.Lock()
	defer cache.segMu.Unlock()

	seg := interfaces.Segment{from, to}
	if ok, overlappedSegment := cache.checkNoOverlap(seg); ok {
		cache.log.Error(fmt.Sprintf("segments is overlapped, from_to is %d_%d, overlapped segment is %d_%d", from, to, overlappedSegment[0], overlappedSegment[1]), "method", "AddSeg")
	}

	cache.segments = append(cache.segments, seg)
	sort.Sort(cache.segments)
}

func (cache *syncCache) DeleteSeg(from, to uint64) {
	cache.segMu.Lock()
	defer cache.segMu.Unlock()

	for index, seg := range cache.segments {
		if seg[0] == from && seg[1] == to {
			cache.segments = append(cache.segments[:index], cache.segments[index+1:]...)
			return
		}
	}

}
func (cache *syncCache) Chunks() interfaces.SegmentList {
	cache.segMu.RLock()
	defer cache.segMu.RUnlock()

	return cache.segments
}

func (cache *syncCache) newDirFd(dirName string) (*os.File, error) {
	var dirFd *os.File
	for dirFd == nil {
		var openErr error
		dirFd, openErr = os.Open(dirName)
		if openErr != nil {
			if os.IsNotExist(openErr) {
				var cErr error
				cErr = os.Mkdir(dirName, 0744)

				if cErr != nil {
					return nil, errors.New(fmt.Sprintf("Create %s failed, error is %s", dirName, cErr.Error()))
				}
			} else {
				return nil, errors.New(fmt.Sprintf("os.Open %s failed, error is %s", dirName, openErr.Error()))
			}
		}
	}

	return dirFd, nil
}

func (cache *syncCache) loadAllSegments() (interfaces.SegmentList, error) {
	allFilename, readErr := cache.dirFd.Readdirnames(0)
	if readErr != nil {
		return nil, errors.New(fmt.Sprintf("cache.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	for _, filename := range allFilename {
		if !cache.isCorrectFile(filename) {
			continue
		}

		segment, err := newSegmentByFilename(filename)
		if err != nil {
			cache.log.Error("newSegmentByFilename failed, error is %s", err)
			continue
		}
		cache.segments = append(cache.segments, segment)
	}

	sort.Sort(cache.segments)
	return cache.segments, nil
}

func (cache *syncCache) isCorrectFile(filename string) bool {
	return strings.HasPrefix(filename, "f_")
}

func (cache *syncCache) toAbsoluteFileName(from, to uint64) string {
	return path.Join(cache.dirName, "f_"+strconv.FormatUint(from, 10)+"_"+strconv.FormatUint(to, 10))
}
