package common

import (
	"fmt"
	"testing"
	"time"
)

func TestCondTimer(t *testing.T) {
	cd := NewCondTimer()

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
			cd.Wait()
			fmt.Printf("wait timeout for %d end,%d \n", i, time.Now().Second())
		}(i)
	}

	cd.Start(time.Second)
	time.Sleep(time.Second * 5)
	cd.Broadcast()
	defer cd.Stop()

}
