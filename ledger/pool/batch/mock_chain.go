package batch

import (
	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
)

var empty types.Hash

func newMockChain() *mockChain {
	return &mockChain{all: make(map[types.Hash]struct{})}
}

type mockChain struct {
	all map[types.Hash]struct{}
}

func (mc mockChain) len() int {
	return len(mc.all)
}

func (mc mockChain) exists(hash types.Hash) error {
	if hash == empty {
		return nil
	}
	_, ok := mc.all[hash]
	if ok {
		return nil
	}
	return errors.New("not exists")
}

func (mc *mockChain) insert(item Item) error {
	keys, _, _ := item.ReferHashes()
	for _, v := range keys {
		_, ok := mc.all[v]
		if ok {
			return errors.Errorf("exist for hash[%s]", v)
		}
		mc.all[v] = struct{}{}
	}
	return nil
}

func (mc *mockChain) execute(p Batch, l Level, bucket Bucket, version uint64) error {
	for _, v := range bucket.Items() {
		err := mc.insert(v)
		if err != nil {
			return err
		}
	}
	return nil
}
