package ledger

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
)

type CompressedFileMeta struct {
	StartHeight uint64
	EndHeight   uint64

	Filename string
	FileSize int64

	BlockNumbers uint64
}

func (f *CompressedFileMeta) Serialize() ([]byte, error) {
	return proto.Marshal(f.Proto())
}

func (f *CompressedFileMeta) Deserialize(buf []byte) error {
	pb := new(vitepb.CompressedFileMeta)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	f.Deproto(pb)

	return nil
}

func (f *CompressedFileMeta) Proto() *vitepb.CompressedFileMeta {
	return &vitepb.CompressedFileMeta{
		StartHeight:  f.StartHeight,
		EndHeight:    f.EndHeight,
		Filename:     f.Filename,
		FileSize:     f.FileSize,
		BlockNumbers: f.BlockNumbers,
	}
}

func (f *CompressedFileMeta) Deproto(pb *vitepb.CompressedFileMeta) {
	f.StartHeight = pb.StartHeight
	f.EndHeight = pb.EndHeight
	f.Filename = pb.Filename
	f.FileSize = pb.FileSize
	f.BlockNumbers = pb.BlockNumbers
}
