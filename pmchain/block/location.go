package chain_block

import "encoding/binary"

const (
	LocationSize = 12
)

type Location struct {
	fileId uint64
	offset uint32
}

func newLocation(fileId uint64, offset uint32) *Location {
	return &Location{
		fileId: fileId,
		offset: offset,
	}
}

func (location *Location) Bytes() []byte {
	fileIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(fileIdBytes, location.fileId)

	offsetBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(offsetBytes, location.offset)

	return append(fileIdBytes, offsetBytes...)
}
func (location *Location) FileId() uint64 {
	return location.fileId
}

func (location *Location) Offset() uint32 {
	return location.offset
}
