package sync_cache

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"sort"
)

func (cache *syncCache) NewWriter(from, to uint64) (io.WriteCloser, error) {
	segment := newSegment(from, to)

	if ok, overlappedSegment := cache.checkNoOverlap(segment); ok {
		return nil, errors.New(fmt.Sprintf("Can't create %d_%d writer, segments is overlapped. Overlapped segment is %d_%d", from, to, overlappedSegment[0], overlappedSegment[1]))
	}

	return nil, nil
}

func (cache *syncCache) checkNoOverlap(segment *Segment) (bool, *Segment) {
	segmentsLength := len(cache.segments)
	n := sort.Search(segmentsLength, func(i int) bool {
		return cache.segments[i][1] >= segment[0]
	})
	if n >= segmentsLength || cache.segments[n][0] <= segment[1] {
		return false, cache.segments[n]
	}

	return true, nil
}
