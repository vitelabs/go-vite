package net

import (
	"bytes"
	"fmt"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/net/vnode"
)

func TestSpeedToString(t *testing.T) {
	speeds := []uint64{
		1000,
		2 * 1024,
		2 * 1024 * 1024,
		2 * 1024 * 1024 * 1024,
		2 * 1024 * 1024 * 1024 * 1024,
	}

	type result struct {
		sf   float64
		unit int
	}

	results := []result{
		{1000, 0},
		{2, 1},
		{2, 2},
		{2, 3},
		{2048, 3},
	}

	for i, speed := range speeds {
		if sf, unit := formatSpeed(float64(speed)); sf != results[i].sf || unit != results[i].unit {
			t.Errorf("wrong speeed: %f, unit: %d", sf, unit)
		} else {
			fmt.Println(speedToString(float64(speed)))
		}
	}
}

func TestSyncHandshakeMsg_Serialize(t *testing.T) {
	var s = syncHandshake{
		id:    vnode.RandomNodeID(),
		key:   make([]byte, 32),
		time:  time.Now().Unix(),
		token: []byte{1, 2, 3},
	}

	data, err := s.Serialize()
	if err != nil {
		panic(err)
	}

	var s2 = &syncHandshake{}
	err = s2.deserialize(data)
	if err != nil {
		panic(err)
	}

	if s.id != s2.id {
		t.Errorf("different id: %s %s", s.id, s2.id)
	}
	if false == bytes.Equal(s.key, s2.key) {
		t.Errorf("different key")
	}
	if s.time != s2.time {
		t.Errorf("different time")
	}
	if false == bytes.Equal(s.token, s2.token) {
		t.Errorf("different token")
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

//func TestCodec(t *testing.T) {
//	const addr = "localhost:8888"
//	ln, err := net2.Listen("tcp", addr)
//	if err != nil {
//		panic(err)
//	}
//
//	messages := []syncMsg{
//		&syncHandshakeMsg{
//			key:  []byte("hello"),
//			time: time.Now(),
//			sign: []byte("world"),
//		},
//		syncHandshakeDone,
//		syncHandshakeErr,
//		&syncRequestMsg{
//			from: 1,
//			to:   10,
//		},
//		&syncReadyMsg{
//			from: 1,
//			to:   10,
//			size: 100,
//		},
//		syncMissing,
//	}
//
//	var wg sync.WaitGroup
//
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//
//		conn, err := ln.Accept()
//		if err != nil {
//			panic(err)
//		}
//		_ = ln.Close()
//
//		codec := &syncCodec{
//			Conn:    conn,
//			builder: syncMsgParser,
//		}
//
//		receive := func(msg syncMsg, i int) error {
//			msg2 := messages[i]
//
//			if msg.code() != msg2.code() {
//				t.Errorf("error message %d", msg.code())
//			}
//
//			if msg.code() == syncHandshake {
//				h := msg.(*syncHandshakeMsg)
//				if string(h.key) != "hello" {
//					return fmt.Errorf("error key: %s", h.key)
//				}
//				if string(h.sign) != "world" {
//					return fmt.Errorf("error sign: %s", h.sign)
//				}
//				if h.time.Unix() != time.Now().Unix() {
//					return errors.New("diff time")
//				}
//			}
//			if msg.code() == syncRequest {
//				h := msg.(*syncRequestMsg)
//				if h.from != 1 || h.to != 10 {
//					return fmt.Errorf("error bound: %d - %d", h.from, h.to)
//				}
//			}
//			if msg.code() == syncReady {
//				h := msg.(*syncReadyMsg)
//				if h.from != 1 || h.to != 10 || h.size != 100 {
//					return fmt.Errorf("error ready: %d - %d - %d", h.from, h.to, h.size)
//				}
//			}
//
//			return nil
//		}
//
//		for i := range messages {
//			msg, err := codec.read()
//			if err != nil {
//				panic(err)
//			}
//
//			if err = receive(msg, i); err != nil {
//				t.Error(err)
//			}
//		}
//
//		_ = conn.Close()
//	}()
//
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//
//		conn, err := net2.Dial("tcp", addr)
//		if err != nil {
//			panic(err)
//		}
//
//		codec := &syncCodec{
//			Conn:    conn,
//			builder: syncMsgParser,
//		}
//
//		for _, msg := range messages {
//			err = codec.write(msg)
//			if err != nil {
//				panic(err)
//			}
//		}
//
//		_ = conn.Close()
//	}()
//
//	wg.Done()
//}

func compare(m1, m2 *syncResponse) error {
	if m1.from != m2.from {
		return fmt.Errorf("different from %d %d", m1.from, m2.from)
	}
	if m1.to != m2.to {
		return fmt.Errorf("different to %d %d", m1.to, m2.to)
	}
	if m1.size != m2.size {
		return fmt.Errorf("different size %d %d", m1.to, m2.to)
	}
	if m1.prevHash != m2.prevHash {
		return fmt.Errorf("different prev hash %s %s", m1.prevHash, m2.prevHash)
	}
	if m2.endHash != m2.endHash {
		return fmt.Errorf("different end hash %s %s", m1.endHash, m2.endHash)
	}

	return nil
}
func TestSyncReadyMsg(t *testing.T) {
	var msg = &syncResponse{
		from:     101,
		to:       200,
		size:     20293,
		prevHash: types.Hash{1},
		endHash:  types.Hash{2},
	}

	data, err := msg.Serialize()
	if err != nil {
		panic(err)
	}

	var msg2 = &syncResponse{}
	err = msg2.deserialize(data)
	if err != nil {
		panic(err)
	}

	if err = compare(msg, msg2); err != nil {
		t.Error(err)
	}
}

func TestSyncRequest_Serialize(t *testing.T) {
	var segment = interfaces.Segment{
		From:     101,
		To:       200,
		Hash:     types.Hash{1},
		PrevHash: types.Hash{2},
	}

	var request = &syncRequest{
		from:     segment.From,
		to:       segment.To,
		prevHash: segment.PrevHash,
		endHash:  segment.Hash,
	}

	data, err := request.Serialize()
	if err != nil {
		panic(err)
	}
	var request2 = &syncRequest{}
	err = request2.deserialize(data)
	if err != nil {
		panic(err)
	}

	if *request2 != *request {
		t.Errorf("different request")
	}
}
