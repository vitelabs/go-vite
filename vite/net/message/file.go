package message

type File struct {
	Name string
	Start uint64
	End uint64
}

type FileList struct {
	Files []*File
	Chunk [][2]uint64	// because files don`t contain the latest snapshotblocks
	Nonce uint64 // use only once
}

func (f *FileList) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *FileList) Deserialize(buf []byte) error {
	panic("implement me")
}

type RequestFile struct {
	Name  string
	Nonce uint64
}

func (f *RequestFile) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *RequestFile) Deserialize(buf []byte) error {
	panic("implement me")
}

type Chunk struct {
	Start, End uint64
}

func (c *Chunk) Serialize() ([]byte, error) {
	panic("implement me")
}

func (c *Chunk) Deserialize(buf []byte) error {
	panic("implement me")
}
