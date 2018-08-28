package vm

import (
	"errors"
	"math/big"
	"testing"
)

func TestStack(t *testing.T) {
	st := newStack()
	data1 := big.NewInt(1)
	st.push(data1)
	if st.len() != 1 || st.peek() != data1 {
		t.Fatalf("stack push error")
	}

	st.peek().SetUint64(2)
	if st.peek().Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("stack push error")
	}

	cpy1 := st.pop()
	if cpy1.Cmp(big.NewInt(2)) != 0 || st.len() != 0 {
		t.Fatalf("stack pop error")
	}

	st.push(big.NewInt(1))
	if st.require(1) != nil || st.require(2) == nil {
		t.Fatalf("stack require method error")
	}

	st.push(big.NewInt(2))
	if st.back(0).Cmp(big.NewInt(2)) != 0 || st.back(1).Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("stack back method error")
	}

	pool := poolOfIntPools.get()
	st.dup(pool, 2)
	if st.len() != 3 || st.peek().Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("stack dup error")
	}

	st.swap(2)
	if st.back(0).Cmp(big.NewInt(2)) != 0 || st.back(1).Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("stack swap error")
	}

	t.Log(st.print())
}

func TestStackTable(t *testing.T) {
	tests := []struct {
		st        *stack
		pop, push int
		err       error
	}{
		{&stack{data: []*big.Int{big.NewInt(1)}}, 1, 1, nil},
		{&stack{data: []*big.Int{}}, 1, 1, errors.New("")},
		{&stack{data: []*big.Int{}}, 0, 2, nil},
		{&stack{data: []*big.Int{big.NewInt(1)}}, 0, 2, nil},
		{&stack{data: []*big.Int{big.NewInt(1)}}, 0, 1024, errors.New("")},
		{&stack{data: []*big.Int{big.NewInt(1)}}, 1, 1024, nil},
		{&stack{data: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}}, 1, 1024, errors.New("")},
		{&stack{data: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}}, 3, 1024, nil},
	}
	for i, test := range tests {
		err := makeStackFunc(test.pop, test.push)(test.st)
		if (err == nil && test.err != nil) || (err != nil && test.err == nil) {
			t.Fatalf("%v th make stack fail", i)
		}
	}
}
