package sync_cache

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path"
	"sort"
)

type syncCache struct {
	log     log15.Logger
	dirName string
	dirFd   *os.File

	maxFileSize int64

	segments SegmentList
}

func NewSyncCache(dataDir string) (SyncCache, error) {
	dirName := path.Join(dataDir, "sync_cache")

	var err error
	cache := &syncCache{
		log:         log15.New("module", "synccache"),
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

func (cache *syncCache) Chunks() SegmentList {
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

func (cache *syncCache) loadAllSegments() (SegmentList, error) {
	allFilename, readErr := cache.dirFd.Readdirnames(0)
	if readErr != nil {
		return nil, errors.New(fmt.Sprintf("cache.dirFd.Readdirnames(0) failed, error is %s", readErr.Error()))
	}

	for _, filename := range allFilename {
		if !cache.isCorrectFile(filename) {
			continue
		}

		segment := newSegmentByFilename(filename)
		cache.segments = append(cache.segments, segment)
	}

	sort.Sort(cache.segments)
	return cache.segments, nil
}

func (cache *syncCache) isCorrectFile(filename string) bool {
	return true
}
