package chain_block

const (
	LocationSize = 12
)

type Location struct {
	fileId uint64
	offset int64
}

func NewLocation(fileId uint64, offset int64) *Location {
	return &Location{
		fileId: fileId,
		offset: offset,
	}
}

func (location *Location) FileId() uint64 {
	return location.fileId
}

func (location *Location) Offset() int64 {
	return location.offset
}

func (location *Location) Compare(a *Location) int {
	if location.fileId < a.fileId {
		return -1
	}
	if location.fileId > a.fileId {
		return 1
	}
	if location.offset < a.offset {
		return -1
	}
	if location.offset > a.offset {
		return 1
	}
	return 0
}
