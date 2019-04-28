package queue

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBlockQueue_Add(t *testing.T) {
	const total = 50
	q := NewBlockQueue(total)

	var wg sync.WaitGroup
	var lock sync.Mutex
	var i int
	for k := 0; k < 5; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				lock.Lock()
				if i < total {
					if q.Add(i) == false {
						t.Error("should add success")
					}
					i++

					lock.Unlock()
				} else {
					lock.Unlock()
					break
				}
			}
		}()
	}

	wg.Wait()
	if q.Size() != total {
		t.Errorf("wrong size: %d", q.Size())
	}

	if q.Add(i) {
		t.Error("should add fail, because queue is full")
	}
}

func TestBlockQueue_Pop(t *testing.T) {
	const total = 5
	q := NewBlockQueue(total)

	blockTimeout := 2 * time.Second
	blocked := time.Now()
	go func() {
		time.Sleep(blockTimeout)
		q.Add(2)
		q.Add(1)
	}()

	v := q.Poll()
	if time.Now().Sub(blocked) < blockTimeout {
		t.Fail()
	}

	if v != 2 {
		t.Fail()
	}
}

func TestBlockQueue_Close(t *testing.T) {
	const total = 5
	q := NewBlockQueue(total)

	pending := make(chan struct{})

	go func() {
		q.Poll()
		pending <- struct{}{}
	}()

	go func() {
		q.Close()
	}()

	<-pending
}

func BenchmarkBlockQueue(b *testing.B) {
	const total = 10
	q := NewBlockQueue(total)

	var amount int
	for i := 0; i < b.N; i++ {
		q.Add(i)
		amount += q.Poll().(int)
	}

	fmt.Println(amount)
}

func BenchmarkChannel(b *testing.B) {
	const total = 10
	channel := make(chan interface{}, total)

	var amount int
	for i := 0; i < b.N; i++ {
		channel <- i
		amount += (<-channel).(int)
	}

	fmt.Println(amount)
}
