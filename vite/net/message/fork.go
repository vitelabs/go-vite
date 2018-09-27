package message

import "github.com/vitelabs/go-vite/ledger"

type Fork struct {
	List []*ledger.HashHeight
}

func (f *Fork) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *Fork) Deserialize(buf []byte) error {
	panic("implement me")
}
