package mock_conn

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMockConn_Read(t *testing.T) {
	c1, c2 := Pipe()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		var buf = []byte("hello world1 hello world2 hello world3 hello world4 hello world5")

		for {
			n, err := c1.Write(buf)
			if err != nil {
				panic(err)
			}
			if n != len(buf) {
				t.Error("too short")
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var buf = make([]byte, 11)

		for {
			n, err := c2.Read(buf)
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s\n", buf[:n])
		}
	}()

	go func() {
		time.AfterFunc(3*time.Second, func() {
			err := c1.Close()
			if err != nil {
				panic(err)
			}

			err = c2.Close()
			if err != nil {
				panic(err)
			}
		})
	}()

	wg.Wait()
}

func TestMockConn_SetReadDeadline(t *testing.T) {
	c1, _ := Pipe()

	err := c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		panic(err)
	}

	var buf = make([]byte, 1024)
	n, err := c1.Read(buf)

	if err == nil || err.Error() != "read timeout" {
		t.Error("should read timeout")
	}
	if n != 0 {
		t.Error("should read 0 bytes")
	}
}
