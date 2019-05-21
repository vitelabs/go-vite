package tree

import (
	"encoding/binary"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

var genesisKnot = genGenesisKnot()
var emptyKnot = genEmptyKnot()

func genGenesisKnot() Knot {
	knot := &mockKnot{}
	knot.prevHash = types.Hash{}
	knot.height = 1
	knot.flag = "genesis"

	knot.hash = knot.computeHash()
	return knot
}

func genEmptyKnot() Knot {
	knot := &mockKnot{}
	knot.prevHash = types.Hash{}
	knot.height = 0
	knot.flag = "empty"

	knot.hash = types.Hash{}
	return knot
}
func newMockKnotByHH(height uint64, hash types.Hash, flag string) Knot {
	knot := &mockKnot{}
	knot.prevHash = hash
	knot.height = height + 1
	knot.flag = flag

	knot.hash = knot.computeHash()
	return knot
}
func newMockKnot(prev Knot, flag string) Knot {
	knot := &mockKnot{}
	knot.prevHash = prev.Hash()
	knot.height = prev.Height() + 1
	knot.flag = flag

	knot.hash = knot.computeHash()
	return knot
}

type mockKnot struct {
	flag     string
	prevHash types.Hash
	height   uint64
	hash     types.Hash
}

func (m mockKnot) Height() uint64 {
	return m.height
}

func (m mockKnot) Hash() types.Hash {
	return m.hash
}

func (m mockKnot) PrevHash() types.Hash {
	return m.prevHash
}

func (m mockKnot) computeHash() types.Hash {
	var source []byte
	// PrevHash
	source = append(source, m.prevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, m.height)
	source = append(source, heightBytes...)

	// flag
	source = append(source, []byte(m.flag)...)

	hash, err := types.BytesToHash(crypto.Hash256(source))
	if err != nil {
		panic(err)
	}
	return hash
}
