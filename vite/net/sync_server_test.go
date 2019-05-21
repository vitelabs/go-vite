package net

import (
	"math/rand"
	net2 "net"
	"sync/atomic"
	"testing"
	"time"
)

func Test_File_Server(t *testing.T) {
	const addr = "localhost:8484"
	fs := newSyncServer(addr, nil, nil)

	if err := fs.start(); err != nil {
		t.Fatal(err)
	}

	var conns int32
	for i := 0; i < 100; i++ {
		go func() {
			conn, err := net2.Dial("tcp", addr)
			if err != nil {
				t.Fatal(err)
			}

			atomic.AddInt32(&conns, 1)

			if rand.Intn(10) > 5 {
				time.Sleep(time.Second)
				conn.Close()
				atomic.AddInt32(&conns, -1)
			}
		}()
	}

	time.Sleep(3 * time.Second)

	if len(fs.sconnMap) != int(conns) {
		t.Fail()
	}

	fs.stop()

	if len(fs.sconnMap) != 0 {
		t.Fail()
	}
}
