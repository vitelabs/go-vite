package net

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand"
	net2 "net"
	"sync"
	"testing"
	"time"
)

func TestSyncHandshakeMsg_Serialize(t *testing.T) {
	var s = syncHandshakeMsg{
		key:  make([]byte, 32),
		time: time.Now(),
		sign: make([]byte, 64),
	}

	for i := 0; i < 100; i++ {
		_, _ = crand.Read(s.key)
		_, _ = crand.Read(s.sign)

		buf, err := s.Serialize()
		if err != nil {
			panic(err)
		}
		fmt.Println(len(buf))
	}
}

func Test_FileConns_Del(t *testing.T) {
	var fs = make(connections, 0, 1)
	fs = fs.del(0)
	fs = fs.del(1)

	fs = append(fs, &syncConn{})
	fs = fs.del(0)
	if len(fs) != 0 {
		t.Fail()
	}

	fs = fs.del(0)
	if len(fs) != 0 {
		t.Fail()
	}
}

func Test_wait(t *testing.T) {
	const total = 10
	wait := make([]int, 0, total)

	count := mrand.Intn(total)

	for i := 0; i < count; i++ {
		wait = append(wait, mrand.Intn(total))
	}

	for i := 0; i < 1000; i++ {
		find := mrand.Intn(total)

		for i, n := range wait {
			if n == find {
				if i != len(wait)-1 {
					copy(wait[i:], wait[i+1:])
				}
				wait = wait[:len(wait)-1]
			}
		}
	}
}

func Test_wait_last(t *testing.T) {
	wait := []int{1, 2, 3, 1, 1, 1}
	total := len(wait)

	for j := 0; j < 4; j++ {
		for i, n := range wait {
			if n == 1 {
				if i != len(wait)-1 {
					copy(wait[i:], wait[i+1:])
				}
				wait = wait[:len(wait)-1]
				break
			}
		}

		if len(wait) != (total - j - 1) {
			fmt.Println(len(wait))
			t.Fail()
		}
	}
}

func Test_wait_all(t *testing.T) {
	wait := []int{1, 1, 1, 1, 1, 1}
	total := len(wait)

	for j := 0; j < 4; j++ {
		for i, n := range wait {
			if n == 1 {
				if i != len(wait)-1 {
					copy(wait[i:], wait[i+1:])
				}
				wait = wait[:len(wait)-1]
				break
			}
		}

		if len(wait) != (total - j - 1) {
			fmt.Println(len(wait))
			t.Fail()
		}
	}
}

func TestCodec(t *testing.T) {
	const addr = "localhost:8888"
	ln, err := net2.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	messages := []syncMsg{
		&syncHandshakeMsg{
			key:  []byte("hello"),
			time: time.Now(),
			sign: []byte("world"),
		},
		syncHandshakeDone,
		syncHandshakeErr,
		&syncRequestMsg{
			from: 1,
			to:   10,
		},
		&syncReadyMsg{
			from: 1,
			to:   10,
			size: 100,
		},
		syncMissing,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		_ = ln.Close()

		codec := &syncCodec{
			Conn:    conn,
			builder: syncMsgParser,
		}

		receive := func(msg syncMsg, i int) error {
			msg2 := messages[i]

			if msg.code() != msg2.code() {
				t.Errorf("error message %d", msg.code())
			}

			if msg.code() == syncHandshake {
				h := msg.(*syncHandshakeMsg)
				if string(h.key) != "hello" {
					return fmt.Errorf("error key: %s", h.key)
				}
				if string(h.sign) != "world" {
					return fmt.Errorf("error sign: %s", h.sign)
				}
				if h.time.Unix() != time.Now().Unix() {
					return errors.New("diff time")
				}
			}
			if msg.code() == syncRequest {
				h := msg.(*syncRequestMsg)
				if h.from != 1 || h.to != 10 {
					return fmt.Errorf("error bound: %d - %d", h.from, h.to)
				}
			}
			if msg.code() == syncReady {
				h := msg.(*syncReadyMsg)
				if h.from != 1 || h.to != 10 || h.size != 100 {
					return fmt.Errorf("error ready: %d - %d - %d", h.from, h.to, h.size)
				}
			}

			return nil
		}

		for i := range messages {
			msg, err := codec.read()
			if err != nil {
				panic(err)
			}

			if err = receive(msg, i); err != nil {
				t.Error(err)
			}
		}

		_ = conn.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, err := net2.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}

		codec := &syncCodec{
			Conn:    conn,
			builder: syncMsgParser,
		}

		for _, msg := range messages {
			err = codec.write(msg)
			if err != nil {
				panic(err)
			}
		}

		_ = conn.Close()
	}()

	wg.Done()
}

//func TestSyncHandshake(t *testing.T) {
//	var mockPeer = &peer{
//		Peer:        nil,
//		peerMap:     sync.Map{},
//		knownBlocks: nil,
//		errChan:     nil,
//		once:        sync.Once{},
//		log:         nil,
//	}
//	peers := newPeerSet()
//	fac := &defaultSyncConnectionFactory{
//		peers: peers,
//	}
//}
