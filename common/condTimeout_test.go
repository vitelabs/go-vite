package common

import (
	"fmt"
	"testing"
	"time"
)

func TestCondTimeout(t *testing.T) {
	cd := NewTimeoutCond()

	for i := 0; i < 10; i++ {
		go func(i int) {
			fmt.Printf("wait for %d\n", i)
			defer fmt.Printf("wait for %d end\n", i)
			cd.Wait()
		}(i)
	}

	for i := 0; i < 5; i++ {
		go func(i int) {
			fmt.Printf("wait timeout for %d, %d\n", i, time.Now().Second())
			cd.WaitTimeout(time.Second * 2)
			fmt.Printf("wait timeout for %d end,%d \n", i, time.Now().Second())
		}(i)
	}

	time.Sleep(time.Second * 5)
	cd.Broadcast()

}
