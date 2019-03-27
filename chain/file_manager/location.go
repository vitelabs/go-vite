package chain_file_manager

const (
	LocationSize = 12
)

type Location struct {
	FileId uint64
	Offset int64
}

func NewLocation(fileId uint64, offset int64) *Location {
	return &Location{
		FileId: fileId,
		Offset: offset,
	}
}

func (location *Location) Compare(a *Location) int {
	if location.FileId < a.FileId {
		return -1
	}
	if location.FileId > a.FileId {
		return 1
	}
	if location.Offset < a.Offset {
		return -1
	}
	if location.Offset > a.Offset {
		return 1
	}
	return 0
}
