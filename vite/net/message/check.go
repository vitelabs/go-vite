package message

import (
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

type HashHeightList struct {
	// from low to high
	Points []*ledger.HashHeight // current height - 75, current height
}

func (c *HashHeightList) Serialize() ([]byte, error) {
	pb := &vitepb.HashHeightList{
		Points: make([]*vitepb.HashHeight, len(c.Points)),
	}

	for i, p := range c.Points {
		pb.Points[i] = p.Proto()
	}

	return proto.Marshal(pb)
}

func (c *HashHeightList) Deserialize(data []byte) error {
	pb := &vitepb.HashHeightList{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	c.Points = make([]*ledger.HashHeight, len(pb.Points))

	for i, p := range pb.Points {
		c.Points[i] = &ledger.HashHeight{}
		err = c.Points[i].DeProto(p)
		if err != nil {
			return err
		}
	}

	return nil
}

type GetHashHeightList struct {
	From *ledger.HashHeight
	Step uint64
	To   uint64
}

func (c *GetHashHeightList) Serialize() ([]byte, error) {
	pb := &vitepb.GetHashHeightList{
		From: c.From.Proto(),
		Step: c.Step,
		To:   c.To,
	}

	return proto.Marshal(pb)
}

func (c *GetHashHeightList) Deserialize(data []byte) (err error) {
	pb := &vitepb.GetHashHeightList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	if pb.From == nil {
		return errors.New("missing from point")
	}

	c.From = &ledger.HashHeight{}
	err = c.From.DeProto(pb.From)
	if err != nil {
		return
	}

	c.Step = pb.Step
	c.To = pb.To

	return nil
}
