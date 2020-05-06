package utils

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRWLock(t *testing.T) {
	var mutex sync.RWMutex
	fmt.Println("------start-------")
	go func() {
		fmt.Println("----rlock---------")
		mutex.RLock()
		fmt.Println("----rlock done---------")
		fmt.Println("----lock---------")
		mutex.Lock()
		fmt.Println("----lock done---------")

		fmt.Println("-------------")
	}()

	time.Sleep(time.Second * 2)

	fmt.Println("----unrlock---------")
	mutex.RUnlock()
	fmt.Println("----unrlock done---------")
	fmt.Println("----unlock---------")
	mutex.Unlock()
	fmt.Println("----unlock done---------")
}

func TestCond(t *testing.T) {
	wait := sync.WaitGroup{}
	locker := new(sync.Mutex)
	cond := sync.NewCond(locker)

	for i := 0; i < 3; i++ {
		go func(i int) {
			defer wait.Done()
			wait.Add(1)
			cond.L.Lock()
			fmt.Println("Waiting start...")
			cond.Wait()
			fmt.Println("Waiting end...")
			cond.L.Unlock()

			fmt.Println("Goroutine run. Number:", i)
		}(i)
	}

	time.Sleep(2e9)
	cond.L.Lock()
	//cond.Signal()
	cond.Broadcast()
	cond.L.Unlock()

	//time.Sleep(2e9)
	//cond.L.Lock()
	//cond.Signal()
	//cond.L.Unlock()
	//
	//time.Sleep(2e9)
	//cond.L.Lock()
	//cond.Signal()
	//cond.L.Unlock()

	wait.Wait()
}

func TestRwMutex(t *testing.T) {
	wait := sync.WaitGroup{}
	locker := new(sync.RWMutex)

	for i := 0; i < 3; i++ {
		go func(i int) {
			defer wait.Done()
			defer locker.RUnlock()
			wait.Add(1)
			fmt.Println("RLock start  get...", i)
			locker.RLock()
			fmt.Println("RLock got...", i)
			time.Sleep(4 * time.Second)
			fmt.Println("RLock end...", i)
			fmt.Println("Goroutine run. Number:", i)
		}(i)
	}

	go func(i int) {
		defer wait.Done()
		defer locker.Unlock()
		wait.Add(1)
		fmt.Println("WLock start get...", i)
		locker.Lock()
		fmt.Println("WLock got...", i)
		time.Sleep(2 * time.Second)
		fmt.Println("WLock end...", i)
		fmt.Println("Goroutine run. Number:", i)
	}(8)
	time.Sleep(1 * time.Second)

	wait.Wait()
}
