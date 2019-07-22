package circle

type TraverseFn = func(key Key, value Value) bool

type Map interface {
	Get(key Key) (Value, bool)
	Put(key Key, value Value)
	Traverse(TraverseFn)
	Size() int
}

type circleMap struct {
	m map[Key]Value
	l List
}

func (cm *circleMap) Size() int {
	return cm.l.Size()
}

func (cm *circleMap) Traverse(fn TraverseFn) {
	cm.l.Traverse(func(key Key) bool {
		if v, ok := cm.m[key]; ok {
			return fn(key, v)
		}
		return true
	})
}

func (cm *circleMap) Get(key Key) (Value, bool) {
	v, ok := cm.m[key]
	return v, ok
}

func (cm *circleMap) Put(key Key, value Value) {
	if _, ok := cm.m[key]; ok {
		cm.m[key] = value
		return
	}

	old := cm.l.Put(key)
	if old != nil {
		delete(cm.m, old)
	}
	cm.m[key] = value
}

func NewMap(total int) Map {
	return &circleMap{
		m: make(map[Key]Value, total),
		l: NewList(total),
	}
}
