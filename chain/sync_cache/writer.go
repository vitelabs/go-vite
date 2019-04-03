package sync_cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interfaces"
	"io"
	"os"
	"sort"
)

func (cache *syncCache) NewWriter(from, to uint64) (io.WriteCloser, error) {
	cache.segMu.RLock()
	cache.segMu.RUnlock()

	segment := newSegment(from, to)

	if ok, overlappedSegment := cache.checkNoOverlap(segment); ok {
		return nil, errors.New(fmt.Sprintf("Can't create %d_%d writer, segments is overlapped. Overlapped segment is %d_%d", from, to, overlappedSegment[0], overlappedSegment[1]))
	}

	return cache.createNewFile(from, to)
}

func (cache *syncCache) checkNoOverlap(segment interfaces.Segment) (bool, *interfaces.Segment) {
	segmentsLength := len(cache.segments)
	n := sort.Search(segmentsLength, func(i int) bool {
		return cache.segments[i][1] >= segment[0]
	})

	if n >= segmentsLength || cache.segments[n][0] <= segment[1] {
		return false, &cache.segments[n]
	}

	return true, nil
}

func (cache *syncCache) createNewFile(from, to uint64) (io.WriteCloser, error) {
	filename := cache.toAbsoluteFileName(from, to)
	file, cErr := os.Create(filename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	cache.AddSeg(from, to)

	return file, nil
}
