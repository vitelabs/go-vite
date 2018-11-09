package trie

import (
	"fmt"
	"sync"
	"testing"
)

type tObject struct {
	a int
}

func (to *tObject) abc() {
	fmt.Println(to.a)
}

func TestThrowErr(t *testing.T) {
	sw := sync.WaitGroup{}

	var mutex sync.Mutex
	for i := 0; i < 100; i++ {
		sw.Add(1)
		go func() {
			defer sw.Done()
			for j := 0; j < 100; j++ {
				if j%3000 == 0 {
					mutex.Lock()
					var tob *tObject
					tob.abc()
					mutex.Unlock()
				}
			}
		}()
	}
	sw.Wait()
}
