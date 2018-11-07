package list

type Element struct {
	Value interface{}
	next  *Element
}

type List struct {
	head  *Element
	tail  *Element
	count int
}

func New() *List {
	head := &Element{}
	return &List{
		head: head,
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
	e := l.head.next
	if e == nil {
		return nil
	}

	l.Remove(l.head, e)
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
	for prev, current := l.head, l.head.next; current != nil; prev, current = current, current.next {
		fn(prev, current)
	}
}

func (l *List) Size() int {
	return l.count
}

func (l *List) Clear() {
	l.head.next = nil
	l.tail.next = nil
	l.count = 0
}
