package blockQueue

import (
	"testing"
	"time"
)

func TestBlockQueue_Push(t *testing.T) {
	q := New()

	q.Push(1)
	q.Push(2)
	if q.Size() != 2 {
		t.Fail()
	}
}

func TestBlockQueue_Pop(t *testing.T) {
	q := New()

	blockTimeout := 2 * time.Second
	blocked := time.Now()
	go func() {
		time.Sleep(blockTimeout)
		q.Push(2)
		q.Push(1)
	}()

	v := q.Pop()
	if time.Now().Sub(blocked) < blockTimeout {
		t.Fail()
	}

	if v != 2 {
		t.Fail()
	}
}

func TestBlockQueue_Close(t *testing.T) {
	q := New()
	pending := make(chan struct{})

	go func() {
		q.Pop()
		pending <- struct{}{}
	}()

	go func() {
		q.Close()
	}()

	<-pending
}
