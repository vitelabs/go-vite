package common

import (
	"fmt"
	"testing"

	"time"
)

func TestGo(t *testing.T) {
	Go(func() {
		fmt.Println("hello world")
		//panic(errors.New("test error"))
		var s *sss

		println(s.aaa)

	})

	time.Sleep(time.Second)
}

type sss struct {
	aaa string
}
