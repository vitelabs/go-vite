package sync_cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interfaces"
	"strconv"
	"strings"
)

func newSegment(from uint64, to uint64) interfaces.Segment {
	return interfaces.Segment{from, to}
}
func newSegmentByFilename(filename string) (interfaces.Segment, error) {
	f1 := strings.Replace(filename, "f_", "", 1)
	strList := strings.Split(f1, "_")
	if len(strList) != 2 {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid.", filename))
	}

	from, err := strconv.ParseUint(strList[0], 10, 64)
	if err != nil {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid. Error: %s", filename, err))
	}
	to, err := strconv.ParseUint(strList[1], 10, 64)
	if err != nil {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid. Error: %s", filename, err))
	}

	return interfaces.Segment{from, to}, nil
}
