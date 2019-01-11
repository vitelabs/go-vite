package list

// Element in list
type Element struct {
	Value interface{}
	next  *Element
}

type List interface {
	Append(data interface{})
	Shift() interface{}
	UnShift(data interface{})
	Remove(prev, current *Element)
	Traverse(handler func(value interface{}) bool)
	Filter(filter func(value interface{}) bool)
	Size() int
	Clear()
}

type list struct {
	head  *Element
	tail  *Element
	count int
}

// New create a linked list
func New() List {
	head := &Element{}
	return &list{
		head: head,
		tail: head,
	}
}

// Append an element at tail of list
func (l *list) Append(data interface{}) {
	e := &Element{
		Value: data,
	}

	l.tail.next = e
	l.tail = e
	l.count++
}

// Shift will remove the first element of list
func (l *list) Shift() interface{} {
	e := l.head.next
	if e == nil {
		return nil
	}

	l.Remove(l.head, e)
	return e.Value
}

// UnShift use to insert an element to front of list
func (l *list) UnShift(data interface{}) {
	e := &Element{
		Value: data,
	}

	e.next = l.head.next
	l.head.next = e

	if e.next == nil {
		l.tail = e
	}

	l.count++
}

// Remove remove an element from list
func (l *list) Remove(prev, current *Element) {
	prev.next = current.next
	if current.next == nil {
		l.tail = prev
	}
	l.count--
}

// Traverse list, if handler return false, will not iterate forward elements
func (l *list) Traverse(handler func(value interface{}) bool) {
	for current := l.head.next; current != nil; current = current.next {
		if !handler(current.Value) {
			break
		}
	}
}

// Filter, if filter return true, will remove those elements
func (l *list) Filter(filter func(value interface{}) bool) {
	for prev, current := l.head, l.head.next; current != nil; {
		if filter(current.Value) {
			// remove
			l.Remove(prev, current)
			current = prev.next
		} else {
			prev, current = current, current.next
		}
	}
}

// Size indicate how many elements in list
func (l *list) Size() int {
	return l.count
}

// Clear remove all elements from list
func (l *list) Clear() {
	l.head.next = nil
	l.tail = l.head
	l.count = 0
}
