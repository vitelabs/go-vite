package pool

type emptyPackage struct {
	ls []*level
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
