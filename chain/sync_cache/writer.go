package sync_cache

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interfaces"
)

func (cache *syncCache) NewWriter(segment interfaces.Segment) (io.WriteCloser, error) {
	cache.segMu.RLock()
	defer cache.segMu.RUnlock()

	if ok := cache.checkOverlap(segment); ok {
		return nil, errors.New(fmt.Sprintf("Can't create %d_%d writer, segments is overlapped.", segment.Bound[0], segment.Bound[1]))
	}

	return cache.createNewFile(segment)
}

func (cache *syncCache) checkOverlap(segment interfaces.Segment) bool {
	segmentsLength := len(cache.segments)
	if segmentsLength <= 0 {
		return false
	}

	n := sort.Search(segmentsLength, func(i int) bool {
		return cache.segments[i].Bound[0] > segment.Bound[1]
	})

	if n == 0 || cache.segments[n-1].Bound[1] < segment.Bound[0] {
		return false
	}

	return true
}

func (cache *syncCache) createNewFile(segment interfaces.Segment) (io.WriteCloser, error) {
	filename := cache.toAbsoluteFileName(segment)
	file, cErr := os.Create(filename)

	if cErr != nil {
		return nil, errors.New("Create file failed, error is " + cErr.Error())
	}

	cache.addSeg(segment)

	return file, nil
}

func (cache *syncCache) addSeg(segment interfaces.Segment) {
	cache.segments = append(cache.segments, segment)
	sort.Sort(cache.segments)
}
