package chain_block

const (
	LocationSize = 12
)

type Location struct {
	fileId uint64
	offset uint32
}

func NewLocation(fileId uint64, offset uint32) *Location {
	return &Location{
		fileId: fileId,
		offset: offset,
	}
}

func (location *Location) FileId() uint64 {
	return location.fileId
}

func (location *Location) Offset() uint32 {
	return location.offset
}
