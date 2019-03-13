package pool

import "github.com/vitelabs/go-vite/common/types"

type emptyPackage struct {
	ls []*level
}

func (self *emptyPackage) Exists(hash types.Hash) bool {
	panic("implement me")
}

func (self *emptyPackage) AddItem(item *Item) error {
	panic("implement me")
}

func (self *emptyPackage) Levels() []Level {
	panic("implement me")
}

func (self *emptyPackage) Size() int {
	panic("implement me")
}

func (self *emptyPackage) Info() string {
	panic("implement me")
}
