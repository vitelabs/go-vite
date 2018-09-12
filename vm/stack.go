package vm

import (
	"fmt"
	"math/big"
)

type stack struct {
	data []*big.Int
}

func newStack() *stack {
	return &stack{data: make([]*big.Int, 0, stackLimit)}
}

func (st *stack) push(d *big.Int) {
	st.data = append(st.data, d)
}

func (st *stack) pop() (ret *big.Int) {
	ret = st.data[len(st.data)-1]
	st.data = st.data[:len(st.data)-1]
	return
}

func (st *stack) peek() *big.Int {
	return st.data[st.len()-1]
}

func (st *stack) len() int {
	return len(st.data)
}

func (st *stack) require(n int) error {
	if st.len() < n {
		return fmt.Errorf("stack underflow (%d <=> %d)", len(st.data), n)
	}
	return nil
}

// back returns the n'th item in stack
func (st *stack) back(n int) *big.Int {
	return st.data[st.len()-n-1]
}

func (st *stack) dup(pool *intPool, n int) {
	st.push(pool.get().Set(st.data[st.len()-n]))
}

func (st *stack) swap(n int) {
	st.data[st.len()-n], st.data[st.len()-1] = st.data[st.len()-1], st.data[st.len()-n]
}

func (st *stack) print() string {
	var result string
	if len(st.data) > 0 {
		for i, val := range st.data {
			if i == len(st.data)-1 {
				result += val.Text(16)
			} else {
				result += val.Text(16) + ", "
			}
		}
	}
	return result
}
