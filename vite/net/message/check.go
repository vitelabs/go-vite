package message

import (
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

var errMissingPoints = errors.New("missing from points")

func pointsToPb(points []*ledger.HashHeight) []*vitepb.HashHeight {
	pb := make([]*vitepb.HashHeight, len(points))
	for i, h := range points {
		pb[i] = h.Proto()
	}

	return pb
}

func pbToPoints(pb []*vitepb.HashHeight) (points []*ledger.HashHeight, err error) {
	points = make([]*ledger.HashHeight, len(pb))

	for i, h := range pb {
		hh := new(ledger.HashHeight)
		err = hh.DeProto(h)
		if err != nil {
			return
		}
		points[i] = hh
	}

	return
}

type HashHeightList struct {
	Points []*ledger.HashHeight // from low to high
}

func (c *HashHeightList) Serialize() ([]byte, error) {
	pb := &vitepb.HashHeightList{
		Points: pointsToPb(c.Points),
	}

	return proto.Marshal(pb)
}

func (c *HashHeightList) Deserialize(data []byte) (err error) {
	pb := &vitepb.HashHeightList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	c.Points, err = pbToPoints(pb.Points)

	return
}

type GetHashHeightList struct {
	From []*ledger.HashHeight // from high to low
	Step uint64
	To   uint64
}

func (c *GetHashHeightList) Serialize() ([]byte, error) {
	pb := &vitepb.GetHashHeightList{
		//From: pointsToPb(c.From),
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

	//if len(pb.From) == 0 {
	//	return errMissingPoints
	//}

	//c.From, err = pbToPoints(pb.From)
	c.Step = pb.Step
	c.To = pb.To

	return
}
