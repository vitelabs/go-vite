package sync_cache

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

var dbItemPrefix = []byte("c:")

type cacheItem struct {
	interfaces.Segment
	done     bool // is write done
	verified bool
	filename string
	size     int64
}

func (c *cacheItem) dbKey() (key []byte) {
	key = make([]byte, 18)

	copy(key, dbItemPrefix)
	binary.BigEndian.PutUint64(key[2:], c.From)
	binary.BigEndian.PutUint64(key[10:], c.To)

	return key
}

func (c *cacheItem) Serialize() ([]byte, error) {
	pb := &vitepb.CacheItem{
		From:     c.From,
		To:       c.To,
		PrevHash: c.PrevHash.Bytes(),
		Hash:     c.Hash.Bytes(),
		Points:   nil,
		Verified: c.verified,
		Filename: c.filename,
		Done:     c.done,
		Size:     c.size,
	}

	if plen := len(c.Points); plen > 0 {
		pb.Points = make([]*vitepb.HashHeight, 0, plen)
		for _, p := range c.Points {
			pb.Points = append(pb.Points, &vitepb.HashHeight{
				Height: p.Height,
				Hash:   p.Hash.Bytes(),
			})
		}
	}

	return proto.Marshal(pb)
}

func (c *cacheItem) DeSerialize(data []byte) (err error) {
	pb := &vitepb.CacheItem{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	c.From = pb.From
	c.To = pb.To
	c.PrevHash, err = types.BytesToHash(pb.PrevHash)
	if err != nil {
		return
	}
	c.Hash, err = types.BytesToHash(pb.Hash)
	if err != nil {
		return
	}
	c.filename = pb.Filename
	c.verified = pb.Verified
	c.done = pb.Done
	c.size = pb.Size

	if plen := len(pb.Points); plen > 0 {
		c.Points = make([]*ledger.HashHeight, 0, plen)
		for _, p := range pb.Points {
			hh := &ledger.HashHeight{
				Height: p.Height,
			}
			hh.Hash, err = types.BytesToHash(p.Hash)
			if err != nil {
				return
			}

			c.Points = append(c.Points, hh)
		}
	}

	return
}

type cacheItems []*cacheItem

func (cs cacheItems) Len() int {
	return len(cs)
}

func (cs cacheItems) Less(i, j int) bool {
	return cs[i].From < cs[j].To
}

func (cs cacheItems) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}
