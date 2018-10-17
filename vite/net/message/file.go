package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
	"strconv"
	"strings"
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

func (f *FileList) String() string {
	var builder strings.Builder
	builder.WriteString("FileList<")

	if len(f.Files) > 0 {
		builder.WriteString("file:")
		builder.WriteString(strconv.FormatUint(f.Files[0].StartHeight, 10))
		builder.WriteString("-")
		builder.WriteString(strconv.FormatUint(f.Files[len(f.Files)-1].EndHeight, 10))
		builder.WriteString("/")
	}

	if len(f.Chunks) > 0 {
		builder.WriteString("chunk:")
		for _, chunk := range f.Chunks {
			builder.WriteString(strconv.FormatUint(chunk[0], 10))
			builder.WriteString("-")
			builder.WriteString(strconv.FormatUint(chunk[1], 10))
		}
	}

	builder.WriteString(">")

	return builder.String()
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

func (f *GetFiles) String() string {
	var builder strings.Builder
	builder.WriteString("GetFiles<")

	for _, file := range f.Names {
		builder.WriteString(file)
	}

	builder.WriteString(">")

	return builder.String()
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

func (c *GetChunk) String() string {
	return "GetChunk<" + strconv.FormatUint(c.Start, 10) + "-" + strconv.FormatUint(c.End, 10) + ">"
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
