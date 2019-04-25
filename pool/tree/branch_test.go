package tree

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranch_GetKnotAndBranch(t *testing.T) {
	root := newMockBranchRoot()

	for i := 0; i < 2; i++ {
		root.addHead(newMockKnot(root.Head(), "root"))
	}

	height, hash := root.HeadHH()

	b1 := newBranch(newBranchBase(height, hash, height, hash, "unittest-1"), root)

	for i := 0; i < 5; i++ {
		h1, h2 := b1.headHH()
		b1.addHead(newMockKnotByHH(h1, h2, "b1"))
	}

	h1, h2 := b1.HeadHH()
	b2 := newBranch(newBranchBase(h1, h2, h1, h2, "unittest-2"), b1)

	for i := 0; i < 10; i++ {
		h1, h2 := b2.headHH()
		b2.addHead(newMockKnotByHH(h1, h2, "b2"))
	}

	assert.Equal(t, uint64(17), b2.headHeight, fmt.Sprintf("%d", b2.headHeight))

	knot, b := b2.GetKnotAndBranch(1)
	assert.NotNil(t, b)
	assert.Equal(t, b.Id(), root.Id())
	assert.NotNil(t, knot)
	assert.Equal(t, knot.Height(), uint64(1))

	knot, b = b2.GetKnotAndBranch(7)
	assert.NotNil(t, b)
	assert.Equal(t, b.Id(), b1.Id())
	assert.NotNil(t, knot)
	assert.Equal(t, knot.Height(), uint64(7))

	knot, b = b2.GetKnotAndBranch(8)
	assert.NotNil(t, b)
	assert.Equal(t, b.Id(), b2.Id())
	assert.NotNil(t, knot)
	assert.Equal(t, knot.Height(), uint64(8))

	knot, b = b2.GetKnotAndBranch(17)
	assert.NotNil(t, b)
	assert.Equal(t, b.Id(), b2.Id())
	assert.NotNil(t, knot)
	assert.Equal(t, knot.Height(), uint64(17))

	knot, b = b2.GetKnotAndBranch(18)
	assert.Nil(t, b)
	assert.Nil(t, knot)

}
