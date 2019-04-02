package consensus_db

import (
	"math/big"

	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
)

type Content struct {
	ExpectedNum uint32
	FactualNum  uint32
}

func (self Content) Copy() *Content {
	return &self
}

func (self *Content) Merge(c *Content) {
	self.ExpectedNum = self.ExpectedNum + c.ExpectedNum
	self.FactualNum = self.FactualNum + c.FactualNum
}

func (self *Content) Rate() int32 {
	if self.ExpectedNum == 0 {
		return -1
	}
	if self.FactualNum == 0 {
		return 0
	}
	result := big.NewInt(0).Div(big.NewInt(int64(self.FactualNum*1000000)), big.NewInt(int64(self.ExpectedNum)))
	return int32(result.Int64())
}
func (self *Content) AddNum(ExpectedNum uint32, FactualNum uint32) {
	self.ExpectedNum = self.ExpectedNum + ExpectedNum
	self.FactualNum = self.FactualNum + FactualNum
}

type Point struct {
	PrevHash *types.Hash
	Hash     *types.Hash
	Sbps     map[types.Address]*Content
}

func (self *Point) Marshal() ([]byte, error) {
	pb := &vitepb.ConsensusPoint{}
	if self.Hash != nil {
		pb.Hash = self.Hash.Bytes()
	}
	if self.PrevHash != nil {
		pb.PrevHash = self.PrevHash.Bytes()
	}
	if len(self.Sbps) > 0 {
		pb.Contents = make([]*vitepb.PointContent, len(self.Sbps))
		i := 0
		for k, v := range self.Sbps {
			c := &vitepb.PointContent{}
			c.Address = k.Bytes()
			c.ENum = v.ExpectedNum
			c.FNum = v.FactualNum

			pb.Contents[i] = c
			i++
		}
	}
	buf, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (self *Point) Unmarshal(buf []byte) error {
	pb := &vitepb.ConsensusPoint{}

	if unmarshalErr := proto.Unmarshal(buf, pb); unmarshalErr != nil {
		return unmarshalErr
	}
	if len(pb.Hash) > 0 {
		self.Hash = &types.Hash{}
		if err := self.Hash.SetBytes(pb.Hash); err != nil {
			return err
		}
	}

	if len(pb.PrevHash) > 0 {
		self.PrevHash = &types.Hash{}
		if err := self.PrevHash.SetBytes(pb.PrevHash); err != nil {
			return err
		}
	}
	self.Sbps = make(map[types.Address]*Content, len(pb.Contents))
	for _, v := range pb.Contents {
		addr := types.Address{}
		if err := addr.SetBytes(v.Address); err != nil {
			return err
		}
		self.Sbps[addr] = &Content{ExpectedNum: v.ENum, FactualNum: v.FNum}
	}
	return nil
}
func (self *Point) Append(p *Point) error {
	if self.Hash == nil {
		self.Hash = p.Hash
		self.PrevHash = p.PrevHash
		self.Sbps = p.Sbps
		return nil
	}

	if self.Hash != p.PrevHash {
		return errors.New("hash and prev hash can't match")
	}

	self.Hash = p.Hash
	self.Sbps = mergeMap(self.Sbps, p.Sbps)
	return nil
}

func mergeMap(m1 map[types.Address]*Content, m2 map[types.Address]*Content) map[types.Address]*Content {
	result := make(map[types.Address]*Content)

	for k, v := range m1 {
		result[k] = v.Copy()
	}
	for k, v := range m2 {
		c, ok := result[k]
		if !ok {
			result[k] = c.Copy()
		} else {
			c.Merge(v)
		}
	}
	return result
}
