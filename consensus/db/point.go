package consensus_db

import (
	"encoding/json"
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

type VoteContent struct {
	Details map[string]*big.Int
	Total   *big.Int
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
	PrevHash types.Hash
	Hash     types.Hash
	Sbps     map[types.Address]*Content

	Votes *VoteContent
}

func (self *Point) Json() string {
	bytes, _ := json.Marshal(self)
	return string(bytes)
}

func (self *Point) Marshal() ([]byte, error) {
	pb := &vitepb.ConsensusPoint{}
	pb.Hash = self.Hash.Bytes()
	pb.PrevHash = self.PrevHash.Bytes()
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

	if self.Votes != nil {
		pb.Votes = make([]*vitepb.PointVoteContent, len(self.Votes.Details)+1)
		i := 0
		c := &vitepb.PointVoteContent{}
		c.VoteCnt = self.Votes.Total.Bytes()
		pb.Votes[i] = c
		i++
		for k, v := range self.Votes.Details {
			c := &vitepb.PointVoteContent{}
			c.Name = k
			c.VoteCnt = v.Bytes()
			pb.Votes[i] = c
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
		if err := self.Hash.SetBytes(pb.Hash); err != nil {
			return err
		}
	}

	if len(pb.PrevHash) > 0 {
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

	if len(pb.Votes) > 0 {
		self.Votes = &VoteContent{Details: make(map[string]*big.Int), Total: big.NewInt(0).SetBytes(pb.Votes[0].VoteCnt)}
		for k, v := range pb.Votes {
			if k == 0 {
				continue
			}
			self.Votes.Details[v.Name] = big.NewInt(0).SetBytes(v.VoteCnt)
		}
	}
	return nil
}
func (self *Point) LeftAppend(p *Point) error {
	if self.Hash == self.PrevHash {
		self.Hash = p.Hash
		self.PrevHash = p.PrevHash
		self.Sbps = p.Sbps
		return nil
	}

	if p.Hash != self.PrevHash {
		return errors.Errorf("hash[%s] and prev[%s] hash can't match", self.Hash, p.PrevHash)
	}

	self.PrevHash = p.PrevHash
	self.Sbps = mergeMap(self.Sbps, p.Sbps)
	return nil
}
func (self *Point) RightAppend(p *Point) error {
	if self.Hash == self.PrevHash {
		self.Hash = p.Hash
		self.PrevHash = p.PrevHash
		self.Sbps = p.Sbps
		return nil
	}

	if self.Hash != p.PrevHash {
		return errors.Errorf("hash[%s] and prev[%s] hash can't match", self.Hash, p.PrevHash)
	}

	self.Hash = p.Hash
	self.Sbps = mergeMap(self.Sbps, p.Sbps)
	return nil
}
func (self *Point) IsEmpty() bool {
	return self.Hash == types.Hash{}
}
func NewEmptyPoint() *Point {
	return &Point{
		PrevHash: types.Hash{},
		Hash:     types.Hash{},
		Sbps:     make(map[types.Address]*Content),
	}
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
