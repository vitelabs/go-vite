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
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package unique_list

import (
	"fmt"
	"strconv"
	"testing"
)

func TestList_Append(t *testing.T) {
	l := New()

	const total = 10
	for i := 0; i < total; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	if l.Size() != total {
		t.Fail()
	}

	i := 0
	l.Traverse(func(key string, value interface{}) bool {
		if v := value.(int); v != i || strconv.Itoa(i) != key {
			t.Fail()
		}
		i++
		return true
	})

	// clear retry
	l.Clear()
	fmt.Printf("%+v\n", l)
	names := []string{
		"foo",
		"bar",
		"foo",
		"hello",
		"bar",
		"world",
	}
	uniqueNames := []string{
		"foo",
		"bar",
		"hello",
		"world",
	}
	uniqueValues := []int{0, 1, 3, 5}

	for i := 0; i < len(names); i++ {
		l.Append(names[i], i)
	}
	if l.Size() != len(uniqueNames) {
		fmt.Println(l.Size())
		t.Fail()
	}

	l.Traverse(func(key string, data interface{}) bool {
		return true
	})

	for i := 0; i < len(uniqueNames); i++ {
		if k, v := l.Shift(); k != uniqueNames[i] || v != uniqueValues[i] {
			t.Fail()
		}
	}
}

func TestList_Shift(t *testing.T) {
	l := New()
	l.Shift()

	const total = 30
	const batch = 10

	for i := 0; i < total; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	for i := 0; i < batch; i++ {
		k, e := l.Shift()
		if i != e.(int) || strconv.Itoa(i) != k {
			fmt.Println("element", i, e)
			t.Fail()
		}
	}

	if l.Size() != (total - batch) {
		fmt.Printf("size %d should be %d\n", l.Size(), total-batch)
		t.Fail()
	}

	for i := total; i < total+batch; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	// shift all elements
	for j := batch; ; j++ {
		if k, v := l.Shift(); v == nil {
			break
		} else if k != strconv.Itoa(j) || v.(int) != j {
			t.Fail()
		}
	}

	if l.Size() != 0 {
		fmt.Println("size should be 0", l.Size())
		t.Fail()
	}

	// append again
	for i := 0; i < total; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	for i := 0; i < batch; i++ {
		k, e := l.Shift()
		if i != e.(int) || k != strconv.Itoa(i) {
			fmt.Println("element", i, e)
			t.Fail()
		}
	}
}

func TestList_UnShift(t *testing.T) {
	l := New()

	const total = 30
	const batch = 10

	for i := 0; i < total; i++ {
		l.UnShift(strconv.Itoa(i), i)
	}

	if l.Size() != total {
		t.Fail()
	}

	for i := 1; i <= batch; i++ {
		k, e := l.Shift()
		if e.(int) != total-i || k != strconv.Itoa(total-i) {
			t.Fail()
		}
	}
}

func TestList_Traverse(t *testing.T) {
	l := New()
	l.Traverse(func(key string, value interface{}) bool {
		return true
	})

	const total = 30

	for i := 0; i < total; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	i := 0
	l.Traverse(func(key string, value interface{}) bool {
		if value.(int) != i || key != strconv.Itoa(i) {
			fmt.Println("traverse fail")
			t.Fail()
		}
		i++
		return true
	})
}

func TestList_Filter(t *testing.T) {
	l := New()
	l.Filter(func(key string, v interface{}) bool {
		return true
	})

	const total = 30

	for i := 0; i < total; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	// remove some elements
	const threshold = 10
	l.Filter(func(key string, value interface{}) bool {
		return value.(int) < threshold
	})

	if l.Size() != total-threshold {
		fmt.Println("remove fail")
		t.Fail()
	}

	// append again
	for i := total; i < total+threshold; i++ {
		l.Append(strconv.Itoa(i), i)
	}

	// rest elements
	i := threshold
	l.Traverse(func(key string, value interface{}) bool {
		if value.(int) != i || key != strconv.Itoa(i) {
			fmt.Println("rest fail")
			t.Fail()
		}
		i++
		return true
	})

	// remove all elements
	l.Filter(func(key string, value interface{}) bool {
		return true
	})
	if l.Size() != 0 {
		fmt.Println("remove all fail")
		t.Fail()
	}

	// append again
	for i := 0; i < total; i++ {
		l.Append(strconv.Itoa(i), i)
	}
	i = 0
	l.Traverse(func(key string, value interface{}) bool {
		if value.(int) != i || key != strconv.Itoa(i) {
			t.Fail()
		}
		i++
		return true
	})
}
