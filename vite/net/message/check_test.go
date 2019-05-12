package message

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/ledger"
)

func compareHashHeightList(c1, c2 *HashHeightList) error {
	if len(c1.Points) != len(c2.Points) {
		return fmt.Errorf("different points length")
	}

	for i, p1 := range c1.Points {
		p2 := c2.Points[i]
		if p1.Height != p2.Height {
			return fmt.Errorf("different point height")
		}
		if p1.Hash != p2.Hash {
			return fmt.Errorf("different point hash %s %s", p1.Hash, p2.Hash)
		}
	}

	return nil
}

func TestHashHeightList_Serialize(t *testing.T) {
	var c = &HashHeightList{}

	for i := 0; i < 5; i++ {
		hh := &ledger.HashHeight{
			Height: uint64(i),
		}
		_, _ = rand.Read(hh.Hash[:])

		c.Points = append(c.Points, hh)
	}

	data, err := c.Serialize()
	if err != nil {
		panic(err)
	}

	var c2 = &HashHeightList{}
	err = c2.Deserialize(data)
	if err != nil {
		panic(err)
	}

	if err = compareHashHeightList(c, c2); err != nil {
		t.Error(err)
	}
}

func compareGetHashHeightList(c1, c2 *GetHashHeightList) error {
	for i, p := range c1.From {
		p2 := c2.From[i]
		if p2.Hash != p.Hash || p2.Height != p.Height {
			return fmt.Errorf("different fep hash: %s/%d %s/%d", p.Hash, p.Height, p2.Hash, p2.Height)
		}
	}

	if c1.Step != c2.Step {
		return fmt.Errorf("different step: %d %d", c1.Step, c2.Step)
	}

	if c1.To != c2.To {
		return fmt.Errorf("different to: %d %d", c1.To, c2.To)
	}

	return nil
}

func TestGetHashHeightList_Serialize(t *testing.T) {
	var c = &GetHashHeightList{
		Step: 100,
		To:   1000,
	}

	data, err := c.Serialize()
	if err != nil {
		panic(err)
	}

	var c2 = &GetHashHeightList{}
	err = c2.Deserialize(data)
	if err == nil {
		panic("error should not be nil")
	}

	var one, two types.Hash
	one[0] = 1
	two[0] = 2
	c.From = []*ledger.HashHeight{
		{100, one},
		{200, two},
	}

	data, err = c.Serialize()
	if err != nil {
		panic(err)
	}

	err = c2.Deserialize(data)
	if err != nil {
		panic(fmt.Sprintf("error should be nil: %v", err))
	}

	if err = compareGetHashHeightList(c, c2); err != nil {
		t.Error(err)
	}
}
