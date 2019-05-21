package circle

type List interface {
	Size() int
	Put(key Key) (old Key)
	Traverse(func(key Key) bool)  // from low to high
	TraverseR(func(key Key) bool) // from high to low
	Reset()
}

type list struct {
	front int
	rear  int
	total int
	l     []Key // has chain zero slot
}

func (l *list) Reset() {
	l.front = 0
	l.rear = 0
}

func (l *list) Traverse(fn func(key Key) bool) {
	for i := l.front; i != l.rear; i = (i + 1) % l.total {
		if !fn(l.l[i]) {
			break
		}
	}
}

func (l *list) TraverseR(fn func(key Key) bool) {
	var index int
	for i := l.rear; i != l.front; i = index {
		index = (i - 1 + l.total) % l.total
		if !fn(l.l[index]) {
			break
		}
	}
}

func (l *list) Size() int {
	return (l.rear - l.front + l.total) % l.total
}

func (l *list) Put(key Key) (replaced Key) {
	// list is full
	if (l.rear+1)%l.total == l.front {
		replaced = l.l[l.front]
		l.front = (l.front + 1) % l.total
	}

	l.l[l.rear] = key
	l.rear = (l.rear + 1) % l.total

	return
}

func NewList(total int) List {
	return &list{
		front: 0,
		rear:  0,
		total: total + 1,
		l:     make([]Key, total+1),
	}
}
