package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

// @section GetSubLedger
// response consist of fileList and chunk
type GetSubLedger = GetSnapshotBlocks

// @section FileList
type FileList struct {
	Files  []*ledger.CompressedFileMeta
	Chunks [][2]uint64 // because files don`t contain the latest snapshotblocks
	Nonce  uint64      // use only once
}

func (f *FileList) Serialize() ([]byte, error) {
	pb := new(vitepb.FileList)

	pb.Nonce = f.Nonce

	length := 2 * len(f.Chunks)
	pb.Chunks = make([]uint64, length)
	for i, c := range f.Chunks {
		pb.Chunks[2*i] = c[0]
		pb.Chunks[2*i+1] = c[1]
	}

	pb.Files = make([]*vitepb.CompressedFileMeta, len(f.Files))
	for i, file := range f.Files {
		pb.Files[i] = file.Proto()
	}

	return proto.Marshal(pb)
}

func (f *FileList) Deserialize(buf []byte) error {
	pb := new(vitepb.FileList)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	f.Nonce = pb.Nonce
	f.Chunks = make([][2]uint64, 0, len(pb.Chunks)/2)
	for i := 0; i < len(pb.Chunks); i += 2 {
		f.Chunks = append(f.Chunks, [2]uint64{pb.Chunks[0], pb.Chunks[1]})
	}
	f.Files = make([]*ledger.CompressedFileMeta, len(pb.Files))
	for i, filePB := range pb.Files {
		file := new(ledger.CompressedFileMeta)
		file.Deproto(filePB)
		f.Files[i] = file
	}

	return nil
}

// @section GetFiles

type GetFiles struct {
	Names []string
	Nonce uint64
}

func (f *GetFiles) Serialize() ([]byte, error) {
	pb := new(vitepb.GetFiles)
	pb.Nonce = f.Nonce
	pb.Names = f.Names
	return proto.Marshal(pb)
}

func (f *GetFiles) Deserialize(buf []byte) error {
	pb := new(vitepb.GetFiles)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	f.Names = pb.Names
	f.Nonce = pb.Nonce

	return nil
}

// @section GetChunk

type GetChunk struct {
	Start, End uint64
}

func (c *GetChunk) Serialize() ([]byte, error) {
	pb := new(vitepb.GetChunk)
	pb.Start = c.Start
	pb.End = c.End
	return proto.Marshal(pb)
}

func (c *GetChunk) Deserialize(buf []byte) error {
	pb := new(vitepb.GetChunk)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	c.Start = pb.Start
	c.End = pb.End
	return nil
}
