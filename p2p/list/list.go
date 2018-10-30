package list

type Element struct {
	Value interface{}
	next  *Element
}

type List struct {
	l     *Element
	tail  *Element
	count int
}

func New() *List {
	head := &Element{}
	return &List{
		l:    head,
		tail: head,
	}
}

func (l *List) Append(data interface{}) {
	e := &Element{
		Value: data,
	}

	l.tail.next = e
	l.tail = e
	l.count++
}

func (l *List) Shift() interface{} {
	e := l.l.next
	if e == nil {
		return nil
	}

	l.Remove(l.l, e)
	return e.Value
}

func (l *List) Remove(prev, current *Element) {
	prev.next = current.next
	if current.next == nil {
		l.tail = prev
	}
	l.count--
}

func (l *List) Traverse(fn func(prev, current *Element)) {
	for prev, current := l.l, l.l.next; current != nil; prev, current = current, current.next {
		fn(prev, current)
	}
}

func (l *List) Size() int {
	return l.count
}
