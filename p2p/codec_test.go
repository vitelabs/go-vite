package p2p

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"net"
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
		i = i << offset

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
		i = i << offset

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
	if msg1.Pid != msg2.Pid {
		return false
	}
	if msg1.Code != msg2.Code {
		return false
	}

	return bytes.Equal(msg1.Payload, msg2.Payload)
}

func TestCodec(t *testing.T) {
	conn1, conn2 := net.Pipe()

	c1 := newTransport(conn1, 100, readMsgTimeout, writeMsgTimeout)
	c2 := newTransport(conn2, 100, readMsgTimeout, writeMsgTimeout)

	const total = 10000
	msgChan := make(chan Msg, total)

	const max = 1<<8 - 1
	t.Run("write", func(t *testing.T) {
		t.Parallel()
		var msg Msg
		var err error
		var buf []byte

		for i := 0; i < total; i++ {
			buf = make([]byte, i)
			_, _ = crand.Read(buf)

			msg = Msg{
				Pid:     ProtocolID(i % max),
				Code:    ProtocolID(i % max),
				Payload: buf,
			}

			err = c1.WriteMsg(msg)

			if err != nil {
				t.Fatalf("write error: %v", err)
			}

			msgChan <- msg
		}
	})

	t.Run("read", func(t *testing.T) {
		t.Parallel()
		var msg, msgC Msg
		var err error

		for i := 0; i < total; i++ {
			msg, err = c2.ReadMsg()

			if err != nil {
				t.Fatalf("read error: %v", err)
			}
			msgC = <-msgChan

			if MsgEqual(msg, msgC) {
				t.Logf("get message %d/%d, %d bytes", msg.Pid, msg.Code, len(msg.Payload))
			} else {
				t.Fatalf("message not equal")
			}
		}
	})
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
