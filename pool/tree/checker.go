package tree

import (
	"fmt"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/go-errors/errors"
)

func CheckTree(t Tree) error {
	diskId := t.Root().Id()
	currentId := t.Main().Id()
	for _, c := range t.Branches() {
		// refer to disk
		if c.Root().Id() == diskId {
			if c.Id() != currentId {
				return errors.New("refer disk")
			} else {
				err := checkHeadTailLink(c, c.Root())
				if err != nil {
					return err
				}
			}
		} else if c.Root().Id() == currentId {
			// refer to current
			err := checkLink(c, c.Root(), true)
			if err != nil {
				return err
			}
		} else {
			err := checkLink(c, c.Root(), false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkHeadTailLink(c1 Branch, c2 Branch) error {
	if c1.Linked(c2) {
		return nil
	}
	return errors.New(fmt.Sprintf("checkHeadTailLink fail. c1:%s, c2:%s, c1Tail:%s, c1Head:%s, c2Tail:%s, c2Head:%s",
		c1.Id(), c2.Id(), c1.SprintTail(), c1.SprintHead(), c2.SprintTail(), c2.SprintHead()))
}
func checkLink(c1 Branch, c2 Branch, refer bool) error {
	tailHeight, tailHash := c1.TailHH()
	block := c2.GetKnot(tailHeight, refer)
	if block == nil {
		return errors.New(fmt.Sprintf("checkLink fail. c1:%s, c2:%s, refer:%t, c1Tail:%s, c1Head:%s, c2Tail:%s, c2Head:%s",
			c1.Id(), c2.Id(), refer,
			c1.SprintTail(), c1.SprintHead(), c2.SprintTail(), c2.SprintHead()))
	} else if block.Hash() != tailHash {
		return errors.New(fmt.Sprintf("checkLink fail. c1:%s, c2:%s, refer:%t, c1Tail:%s, c1Head:%s, c2Tail:%s, c2Head:%s, hash[%s-%s]",
			c1.Id(), c2.Id(), refer,
			c1.SprintTail(), c1.SprintHead(), c2.SprintTail(), c2.SprintHead(), block.Hash(), tailHash))
	}
	return nil
}

func CheckTreeSize(t Tree) error {
	size := t.Size()

	m := make(map[types.Hash]bool)
	bs := t.Branches()
	for _, v := range bs {
		if v.Type() == Disk {
			return errors.New("contains disk chain")
		}

		b := v.(*branch)
		if b.size() != b.storeSize() {
			return errors.New("branch size is not equals")
		}
		for _, bv := range b.heightBlocks {
			m[bv.Hash()] = true
		}
	}

	if size != uint64(len(m)) {
		return errors.New("tree size is not equals")
	}
	return nil
}
