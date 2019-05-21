package bytes_pool

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestBytesPool_Get(t *testing.T) {
	for i := 0; i < bufSize3; i++ {
		buf := Get(i)
		if len(buf) < i {
			t.Errorf("too small: %d", len(buf))
		}
	}
}

func TestNobytesPool(t *testing.T) {
	const times = 1000000
	const total = 20
	const curr = 5

	var wg sync.WaitGroup

	ch := make(chan []byte, total)

	wg.Add(curr)
	for i := 0; i < curr; i++ {
		go func() {
			defer wg.Done()

			for buf := range ch {
				time.Sleep(time.Millisecond)
				_ = len(buf)
			}
		}()
	}

	var alloc int
	allocFn := func() []byte {
		n := rand.Intn(50000) + 50000
		alloc++
		return make([]byte, n)
	}

	var m runtime.MemStats
	for i := 0; i < times; i++ {
		ch <- allocFn()

		if i%1000 == 0 {
			runtime.ReadMemStats(&m)

			fmt.Printf("%d,%d,%d,%d,%d\n", m.HeapSys, m.HeapAlloc,
				m.HeapIdle, m.HeapReleased, alloc)
		}
	}

	close(ch)
	wg.Wait()
}

func TestBytesPool(t *testing.T) {
	const times = 1000000
	const total = 20
	const curr = 5

	var wg sync.WaitGroup

	ch := make(chan []byte, total)
	wg.Add(curr)
	for i := 0; i < curr; i++ {
		go func() {
			defer wg.Done()

			for buf := range ch {
				time.Sleep(time.Millisecond)
				Put(buf)
			}
		}()
	}

	var m runtime.MemStats
	for i := 0; i < times; i++ {
		n := rand.Intn(50000) + 50000
		ch <- Get(n)

		if i%1000 == 0 {
			runtime.ReadMemStats(&m)

			fmt.Printf("%d,%d,%d,%d,%d\n", m.HeapSys, m.HeapAlloc,
				m.HeapIdle, m.HeapReleased, i)
		}
	}

	close(ch)
	wg.Wait()
}
