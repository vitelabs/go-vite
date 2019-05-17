package message

import (
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

var errMissingPoints = errors.New("missing from points")
var errNilPoint = errors.New("nil HashHeightPoint")

func pointsToPb(points []*HashHeightPoint) []*vitepb.HashHeightPoint {
	pb := make([]*vitepb.HashHeightPoint, len(points))
	for i, h := range points {
		pb[i] = h.Proto()
	}

	return pb
}

func pbToPoints(pb []*vitepb.HashHeightPoint) (points []*HashHeightPoint, err error) {
	points = make([]*HashHeightPoint, len(pb))

	var j int
	for _, h := range pb {
		if h == nil {
			return nil, errNilPoint
		}

		hh := new(HashHeightPoint)
		err = hh.DeProto(h)
		if err != nil {
			return
		}
		points[j] = hh
		j++
	}

	return
}

func hashHeightToPbs(points []*ledger.HashHeight) []*vitepb.HashHeight {
	pb := make([]*vitepb.HashHeight, len(points))
	for i, h := range points {
		pb[i] = h.Proto()
	}

	return pb
}

func pbsToHashHeight(pb []*vitepb.HashHeight) (points []*ledger.HashHeight, err error) {
	points = make([]*ledger.HashHeight, len(pb))

	var j int
	for _, h := range pb {
		if h == nil {
			return nil, errNilPoint
		}

		hh := new(ledger.HashHeight)
		err = hh.DeProto(h)
		if err != nil {
			return
		}
		points[j] = hh
		j++
	}

	return
}

type HashHeightPoint struct {
	ledger.HashHeight
	Size uint64
}

func (p *HashHeightPoint) Proto() *vitepb.HashHeightPoint {
	return &vitepb.HashHeightPoint{
		Point: &vitepb.HashHeight{
			Hash:   p.Hash.Bytes(),
			Height: p.Height,
		},
		Size: p.Size,
	}
}

func (p *HashHeightPoint) DeProto(pb *vitepb.HashHeightPoint) (err error) {
	if pb == nil || pb.Point == nil {
		return errNilPoint
	}

	p.Hash, err = types.BytesToHash(pb.Point.Hash)
	if err != nil {
		return
	}
	p.Height = pb.Point.Height
	p.Size = pb.Size

	return
}

type HashHeightPointList struct {
	Points []*HashHeightPoint // from low to high
}

func (c *HashHeightPointList) Serialize() ([]byte, error) {
	pb := &vitepb.HashHeightList{
		Points: pointsToPb(c.Points),
	}

	return proto.Marshal(pb)
}

func (c *HashHeightPointList) Deserialize(data []byte) (err error) {
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
		From: hashHeightToPbs(c.From),
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

	if len(pb.From) == 0 {
		return errMissingPoints
	}

	c.From, err = pbsToHashHeight(pb.From)
	c.Step = pb.Step
	c.To = pb.To

	return
}
