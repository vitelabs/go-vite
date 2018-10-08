package message

type File struct {
	Name  string
	Start uint64
	End   uint64
}

func (f *File) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *File) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section GetSubLedger
// response consist of fileList and chunk
type GetSubLedger = GetSnapshotBlocks

// @section FileList
type FileList struct {
	Files []*File
	Chunk [][2]uint64 // because files don`t contain the latest snapshotblocks
	Nonce uint64      // use only once
}

func (f *FileList) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *FileList) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section GetFiles

type GetFiles struct {
	Names []string
	Nonce uint64
}

func (f *GetFiles) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *GetFiles) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section GetChunk

type GetChunk struct {
	Start, End uint64
}

func (c *GetChunk) Serialize() ([]byte, error) {
	panic("implement me")
}

func (c *GetChunk) Deserialize(buf []byte) error {
	panic("implement me")
}
