package ledger

type CompressedFileMeta struct {
	StartHeight uint64
	EndHeight   uint64

	Filename string
	FileSize int64

	BlockNumbers uint64
}
