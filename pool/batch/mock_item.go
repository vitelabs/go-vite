package batch

import (
	"encoding/binary"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

type mockItem struct {
	prevHash types.Hash
	hash     types.Hash
	height   uint64
	addr     *types.Address

	keys      []types.Hash
	accBlocks []types.Hash
	sBlock    *types.Hash

	expectedErr error
}

func (m mockItem) ReferHashes() ([]types.Hash, []types.Hash, *types.Hash) {
	return m.keys, m.accBlocks, m.sBlock
}

func (m mockItem) Owner() *types.Address {
	return m.addr
}

func (m mockItem) Hash() types.Hash {
	return m.hash
}

func (m mockItem) Height() uint64 {
	return m.height
}

func (m mockItem) PrevHash() types.Hash {
	return m.prevHash
}

func (m mockItem) computeHash() types.Hash {
	var source []byte
	// PrevHash
	source = append(source, m.prevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, m.height)
	source = append(source, heightBytes...)

	if m.addr != nil {
		// flag
		source = append(source, m.addr.Bytes()...)
	}

	hash, err := types.BytesToHash(crypto.Hash256(source))
	if err != nil {
		panic(err)
	}
	return hash
}

// mock receive item
func NewMockReceiveBlcok(prev Item, sendHash types.Hash) *mockItem {
	item := &mockItem{}

	item.prevHash = prev.Hash()
	item.height = prev.Height() + 1
	item.addr = prev.Owner()
	item.hash = item.computeHash()

	item.keys = append(item.keys, item.hash)

	item.accBlocks = append(item.accBlocks, item.prevHash)
	item.accBlocks = append(item.accBlocks, sendHash)

	return item

}

// mock send item
func NewMockSendBlcok(prev Item) *mockItem {
	item := &mockItem{}

	item.prevHash = prev.Hash()
	item.height = prev.Height() + 1
	item.addr = prev.Owner()
	item.hash = item.computeHash()

	item.keys = append(item.keys, item.hash)
	item.accBlocks = append(item.accBlocks, item.prevHash)

	return item
}

// mock snapshot item
func NewMockSnapshotBlock(prev Item, accBlocks []Item) *mockItem {
	item := &mockItem{}

	item.prevHash = prev.Hash()
	item.height = prev.Height() + 1
	item.addr = prev.Owner()
	item.hash = item.computeHash()
	item.keys = append(item.keys, item.hash)
	for _, v := range accBlocks {
		item.accBlocks = append(item.accBlocks, v.Hash())
	}
	item.sBlock = &item.prevHash
	return item
}

func NewGenesisBlock(addr *types.Address) *mockItem {
	item := &mockItem{}

	item.prevHash = types.Hash{}
	item.height = types.GenesisHeight
	item.addr = addr
	item.hash = item.computeHash()

	item.keys = append(item.keys, item.hash)

	if addr == nil {
		item.sBlock = &item.prevHash
	} else {
		item.accBlocks = append(item.accBlocks, item.prevHash)
	}
	return item
}
