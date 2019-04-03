package message

import (
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
)

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
