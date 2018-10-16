package common

import (
	"fmt"
	"testing"

	"time"

	"github.com/pkg/errors"
)

func TestGo(t *testing.T) {
	Go(func() {
		fmt.Println("hello world")
		panic(errors.New("test error"))
	})

	time.Sleep(time.Second)
}
