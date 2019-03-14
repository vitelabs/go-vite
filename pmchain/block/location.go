package chain_block

import "encoding/binary"

const (
	LocationSize = 12
)

type Location [LocationSize]byte

func newLocation(fileId uint64, offset uint32) *Location {
	fileIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(fileIdBytes, fileId)

	offsetBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(offsetBytes, offset)

	var location Location
	copy(location[:8], fileIdBytes)
	copy(location[8:], offsetBytes)
	return &location
}
