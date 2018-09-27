package message

import (
	"github.com/vitelabs/go-vite/common/types"
)

type HandShake struct {
	CmdSet      uint64
	Height       uint64
	Port         uint16
	Current types.Hash
	Genesis types.Hash
}

func (st *HandShake) Serialize() ([]byte, error) {
	return nil, nil
}

func (st *HandShake) Deserialize(data []byte) error {
	return nil
}
