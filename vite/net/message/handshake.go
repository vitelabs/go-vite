package message

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

type HandShake struct {
	Height      uint64
	Head        types.Hash
	Genesis     types.Hash
	FileAddress vnode.EndPoint
}

func (h *HandShake) Serialize() ([]byte, error) {
	// todo

	return nil, nil
}

func (h *HandShake) Deserialize(data []byte) error {

	return nil
}
