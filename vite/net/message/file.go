package message

type File struct {
	Name string
	Start uint64
	End uint64
}

type FileList struct {
	Files []*File
	Start uint64 // start and end means need query blocks from chainDB
	End   uint64 // because files don`t contain the latest snapshotblocks
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
