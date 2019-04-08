package sync_cache

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interfaces"
)

func (cache *syncCache) NewWriter(from, to uint64) (io.WriteCloser, error) {
	cache.segMu.RLock()
	defer cache.segMu.RUnlock()

	segment := newSegment(from, to)

	if ok := cache.checkOverlap(segment); ok {
		return nil, errors.New(fmt.Sprintf("Can't create %d_%d writer, segments is overlapped.", from, to))
	}

	return cache.createNewFile(from, to)
}

func (cache *syncCache) checkOverlap(segment interfaces.Segment) bool {
	segmentsLength := len(cache.segments)
	if segmentsLength <= 0 {
		return false
	}

	n := sort.Search(segmentsLength, func(i int) bool {
		return cache.segments[i][0] > segment[1]
	})

	if n == 0 || cache.segments[n-1][1] < segment[0] {
		return false
	}

	return true
}

func (cache *syncCache) createNewFile(from, to uint64) (io.WriteCloser, error) {
	filename := cache.toAbsoluteFileName(from, to)
	file, cErr := os.Create(filename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	cache.addSeg(from, to)

	return file, nil
}

func (cache *syncCache) addSeg(from, to uint64) {
	seg := interfaces.Segment{from, to}

	cache.segments = append(cache.segments, seg)
	sort.Sort(cache.segments)
}
