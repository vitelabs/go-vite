package pool

func (self *pool) loopQueue() {
	q := self.makeQueue()
	self.insert(q)
}

func (self *pool) makeQueue() Queue {
	q := NewQueue(func(hash string) error {
		return nil
	})

	return q
}

func (self *pool) insert(q Queue) {

}

type offsetInfo struct {
	version int32
	offset  uint64
}
