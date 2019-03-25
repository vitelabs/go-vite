/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received chain copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package unique_list

// element in list
type element struct {
	key  string
	next *element
}

type UniqueList interface {
	Append(key string, data interface{})
	Shift() (key string, data interface{})
	UnShift(key string, data interface{})
	Traverse(handler func(key string, data interface{}) bool)
	Filter(filter func(key string, data interface{}) bool)
	Size() int
	Clear()
}

type unique_list struct {
	m    map[string]interface{}
	head *element
	tail *element
}

// New create chain linked list
func New() UniqueList {
	head := &element{}
	return &unique_list{
		m:    make(map[string]interface{}),
		head: head,
		tail: head,
	}
}

// Append an element at tail of list
func (l *unique_list) Append(key string, data interface{}) {
	if _, ok := l.m[key]; ok {
		return
	}

	l.m[key] = data
	e := &element{
		key: key,
	}

	l.tail.next = e
	l.tail = e
}

// Shift will remove the first element of list
func (l *unique_list) Shift() (key string, data interface{}) {
	e := l.head.next
	if e == nil {
		return
	}

	v := l.m[e.key]

	l.remove(l.head, e)

	return e.key, v
}

// UnShift use to insert an element to front of list
func (l *unique_list) UnShift(key string, data interface{}) {
	if _, ok := l.m[key]; ok {
		return
	}

	l.m[key] = data

	e := &element{
		key: key,
	}

	e.next = l.head.next
	l.head.next = e
}

// Remove remove an element from list
func (l *unique_list) remove(prev, current *element) {
	prev.next = current.next
	if current.next == nil {
		l.tail = prev
	}

	delete(l.m, current.key)
}

// Traverse list, if handler return false, will not iterate forward elements
func (l *unique_list) Traverse(handler func(key string, value interface{}) bool) {
	for current := l.head.next; current != nil; current = current.next {
		if !handler(current.key, l.m[current.key]) {
			break
		}
	}
}

// Filter, if filter return true, will remove those elements
func (l *unique_list) Filter(filter func(key string, value interface{}) bool) {
	for prev, current := l.head, l.head.next; current != nil; {
		if filter(current.key, l.m[current.key]) {
			// remove
			l.remove(prev, current)
			current = prev.next
		} else {
			prev, current = current, current.next
		}
	}
}

// Size indicate how many elements in list
func (l *unique_list) Size() int {
	return len(l.m)
}

// Clear remove all elements from list
func (l *unique_list) Clear() {
	l.m = make(map[string]interface{})
	l.head.next = nil
	l.tail = l.head
}
