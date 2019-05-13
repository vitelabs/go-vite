package sync_cache

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func NewSegment(from uint64, to uint64, prevHash, hash types.Hash) interfaces.Segment {
	return interfaces.Segment{
		Bound:    [2]uint64{from, to},
		Hash:     hash,
		PrevHash: prevHash,
	}
}

func newSegmentByFilename(filename string) (interfaces.Segment, error) {
	suffix := path.Ext(filename)
	filename = strings.TrimSuffix(filename, suffix)

	f1 := strings.Replace(filename, "f_", "", 1)
	strList := strings.Split(f1, "_")
	if len(strList) != 4 {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid.", filename))
	}

	from, err := strconv.ParseUint(strList[0], 10, 64)
	if err != nil {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid. Error: %s", filename, err))
	}

	prevHash, err := types.HexToHash(strList[1])
	if err != nil {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid. Error: %s", filename, err))
	}

	to, err := strconv.ParseUint(strList[2], 10, 64)
	if err != nil {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid. Error: %s", filename, err))
	}
	hash, err := types.HexToHash(strList[3])
	if err != nil {
		return interfaces.Segment{}, errors.New(fmt.Sprintf("%s is invalid. Error: %s", filename, err))
	}

	return NewSegment(from, to, prevHash, hash), nil
}
