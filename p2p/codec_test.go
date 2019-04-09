package p2p

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestPutUint24(t *testing.T) {
	const maxLen = 8
	buf := make([]byte, maxLen)

	var i uint = 0

	m := PutVarint(buf, i, maxLen)
	if m != 1 {
		t.Fatalf("0 should be 1 byte, but not %d bytes", m)
	}

	i = 1
	for offset := uint(0); offset < maxLen*8; offset++ {
		i = 1 << offset

		length := int(offset/8 + 1)

		m = PutVarint(buf, i, length)

		if m > length {
			t.Fatalf("%d should be %d bytes, but not %d bytes", i, length, m)
		}
	}
}

func TestUint24(t *testing.T) {
	const maxLen = 8
	buf := make([]byte, maxLen)

	var i uint = 0
	m := PutVarint(buf, i, maxLen)
	i2 := Varint(buf[:m])
	if i2 != i {
		t.Fatalf("put 0, but not get %d", i2)
	}

	i = 1
	for offset := uint(0); offset < maxLen*8; offset++ {
		i = 1 << offset

		length := int(offset/8 + 1)

		m = PutVarint(buf, uint(i), length)

		i2 = Varint(buf[:m])
		if i2 != uint(i) {
			t.Fatalf("put %d, but get %d", i, i2)
		}
	}
}

func ExampleSHA3_Write() {
	hash := sha3.New256()

	buf := make([]byte, 10)
	_, err := crand.Read(buf)
	if err != nil {
		panic(err)
	}

	// write change the internal state, will get different sum result
	hash.Write(buf)
	hash1 := hash.Sum(nil)

	hash.Write(buf)
	hash2 := hash.Sum(nil)

	fmt.Println(bytes.Equal(hash1, hash2))
	// Output:
	// false
}

func ExampleSHA3_Reset() {
	hash := sha3.New256()
	hash1 := hash.Sum(nil)

	hash.Write([]byte("hello"))
	hash2 := hash.Sum(nil)

	hash.Reset() // Reset will clean the internal state
	hash3 := hash.Sum(nil)

	hash.Write([]byte("hello"))
	hash4 := hash.Sum(nil)

	fmt.Println(bytes.Equal(hash1, hash3), bytes.Equal(hash2, hash4))
	fmt.Println(bytes.Equal(hash1, hash2))
	// Output:
	// true true
	// false
}

func ExampleSHA3_Sum() {
	hash := sha3.New256()
	hash.Write([]byte("hello"))
	hash.Write([]byte("world"))
	hash1 := hash.Sum(nil)

	hash2 := hash.Sum([]byte("world2")) // append hash1 to `world2`
	hash3 := append([]byte("world2"), hash1...)

	fmt.Println(bytes.Equal(hash2, hash3))
	// Output:
	// true
}

func ExampleSHA3_Size() {
	hash := sha3.New256()
	hash.Write([]byte("hello"))
	size1 := hash.Size()
	hash1 := hash.Sum(nil)
	size2 := hash.Size()

	hash2 := hash.Sum([]byte("world"))
	size3 := hash.Size()

	fmt.Println(len(hash1) == size1)
	fmt.Println(size1 == size2, size1 == size3)
	fmt.Println(size1+5 == len(hash2))
	// Output:
	// true
	// true true
	// true
}

func MsgEqual(msg1, msg2 Msg) bool {
	if msg1.pid != msg2.pid {
		return false
	}
	if msg1.Code != msg2.Code {
		return false
	}
	if msg1.Id != msg2.Id {
		return false
	}

	return bytes.Equal(msg1.Payload, msg2.Payload)
}

func TestCodec(t *testing.T) {
	const addr = "127.0.0.1:10000"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	const total = 1000000
	msgChan := make(chan Msg, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, e := ln.Accept()
		if e != nil {
			panic(err)
		}

		c1 := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		var msg Msg
		var err error
		var buf []byte

		const max = 1 << 8
		for i := uint32(0); i < total; i *= 2 {
			buf = make([]byte, i)
			_, _ = crand.Read(buf)

			msg = Msg{
				pid:     ProtocolID(i % max),
				Code:    ProtocolID(i % max),
				Id:      i,
				Payload: buf,
			}

			msgChan <- msg

			err = c1.WriteMsg(msg)
			fmt.Printf("write message %d/%d, %d bytes\n", msg.pid, msg.Code, len(msg.Payload))

			if err != nil {
				t.Fatalf("write error: %v", err)
			}

			i++
		}

		close(msgChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var msg, msgC Msg
		var err error

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}

		c2 := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		for msgC = range msgChan {
			msg, err = c2.ReadMsg()

			if err != nil {
				if err == io.EOF {
					continue
				}
				t.Fatalf("read error: %v", err)
			}

			if MsgEqual(msg, msgC) {
				fmt.Printf("read message %d/%d, %d bytes\n", msg.pid, msg.Code, len(msg.Payload))
			} else {
				t.Fatalf("message not equal")
			}
		}
	}()

	err = http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		panic(err)
	}

	wg.Wait()
}

/*
// will panic
func TestParallel(t *testing.T) {
	var m = make(map[string]int)

	t.Run("write", func(t *testing.T) {
		t.Parallel() // will run this test parallel
		for i := 0; i < 1000; i++ {
			m["hello"] = i
			time.Sleep(10 * time.Millisecond)
		}
	})

	t.Run("read", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 1000; i++ {
			t.Log(m["hello"])
			time.Sleep(10 * time.Millisecond)
		}
	})
}
*/
