package common

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"
)

func TestRwLock(t *testing.T) {
	//远程获取pprof数据
	go func() {
		log.Println(http.ListenAndServe("localhost:8070", nil))
	}()

	mutex := sync.RWMutex{}

	//ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)
	//ch4 := make(chan int)

	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		<-ch3
		mutex.RLock()
		ch2 <- 1

		time.Sleep(30 * time.Millisecond)
		mutex.RLock()
		defer mutex.RUnlock()
		defer mutex.RUnlock()

		wg.Done()
	}()

	go func() {
		<-ch2
		mutex.Lock()

		defer mutex.Unlock()
		wg.Done()
	}()

	ch3 <- 1
	wg.Wait()
	fmt.Println("-------")
}
