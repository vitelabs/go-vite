package pool

import (
	"encoding/binary"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

var genesisBlock = genGenesisBlock()
var emptyBlock = genEmptyBlock()

func genGenesisBlock() commonBlock {
	knot := &mockCommonBlock{}
	knot.prevHash = types.Hash{}
	knot.height = 1
	knot.flag = "genesis"

	knot.hash = knot.computeHash()
	return knot
}

func genEmptyBlock() commonBlock {
	knot := &mockCommonBlock{}
	knot.prevHash = types.Hash{}
	knot.height = 0
	knot.flag = "empty"

	knot.hash = types.Hash{}
	return knot
}
func newMockCommonBlockByHH(height uint64, hash types.Hash, flag string) commonBlock {
	knot := &mockCommonBlock{}
	knot.prevHash = hash
	knot.height = height + 1
	knot.flag = flag

	knot.hash = knot.computeHash()
	return knot
}
func newMockCommonBlock(prev commonBlock, flag string) commonBlock {
	knot := &mockCommonBlock{}
	knot.prevHash = prev.Hash()
	knot.height = prev.Height() + 1
	knot.flag = flag

	knot.hash = knot.computeHash()
	return knot
}

type mockCommonBlock struct {
	flag     string
	prevHash types.Hash
	height   uint64
	hash     types.Hash
}

func (m mockCommonBlock) Height() uint64 {
	return m.height
}

func (m mockCommonBlock) Hash() types.Hash {
	return m.hash
}

func (m mockCommonBlock) PrevHash() types.Hash {
	return m.prevHash
}

func (*mockCommonBlock) checkForkVersion() bool {
	return false
}

func (*mockCommonBlock) resetForkVersion() {
}

func (*mockCommonBlock) forkVersion() uint64 {
	return 0
}

func (*mockCommonBlock) Source() types.BlockSource {
	return types.Local
}

func (*mockCommonBlock) Latency() time.Duration {
	return time.Second
}

func (*mockCommonBlock) ShouldFetch() bool {
	return false
}

func (*mockCommonBlock) ReferHashes() ([]types.Hash, []types.Hash, *types.Hash) {
	panic("implement me")
}

func (m mockCommonBlock) computeHash() types.Hash {
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
