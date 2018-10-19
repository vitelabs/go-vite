package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
)

func mockPayload() ([]byte, int) {
	var size int

	for {
		size = rand.Intn(10000)
		if size > 0 {
			break
		}
	}

	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(rand.Intn(254))
	}

	return buf, size
}

func mockMsg() (*Msg, error) {
	msg := NewMsg()

	msg.CmdSet = rand.Uint64()
	msg.Cmd = rand.Uint32()
	msg.Id = rand.Uint64()

	payload, size := mockPayload()
	msg.Payload = payload
	msg.Size = uint32(size)

	return msg, nil
}

func TestAsyncMsgConn_Start(t *testing.T) {
	errch := make(chan error)
	const addr = "localhost:8888"

	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			t.Error(err)
		}

		conn, err := ln.Accept()
		if err != nil {
			t.Error(err)
		}

		if err = handleConn(conn, "server"); err != nil {
			t.Error(err)
			errch <- err
		}
	}()

	go func() {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Error(err)
		}

		if err = handleConn(conn, "client"); err != nil {
			t.Error(err)
			errch <- err
		}
	}()

	fmt.Println("test fail", <-errch)
}

func handleConn(conn net.Conn, mark string) error {
	errch := make(chan error)

	c := NewAsyncMsgConn(conn)

	c.handler = func(msg *Msg) {
		fmt.Println(mark, "receive", msg.CmdSet, msg.Cmd, msg.Id, msg.Size)
	}

	c.Start()

	go func() {
		select {
		case err := <-c.errch:
			fmt.Println(mark, "error", err)
			errch <- err
		}
	}()

	for {
		msg, err := mockMsg()
		if err != nil {
			continue
		}

		err = c.SendMsg(msg)
		if err != nil {
			fmt.Println(mark, "send error", err)
			errch <- err
		} else {
			fmt.Println(mark, "send", msg.CmdSet, msg.Cmd, msg.Id, msg.Size)
		}
	}

	err := <-errch
	c.Close()
	return err
}

func TestAsyncMsgConn_Error(t *testing.T) {
	errch := make(chan error)
	const addr = "localhost:8890"

	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			t.Error(err)
		}

		conn, err := ln.Accept()
		if err != nil {
			t.Error(err)
		}

		if err = handleConn(conn, "server"); err != nil {
			t.Error(err)
			errch <- err
		}
	}()

	go func() {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Error(err)
		}

		if err = handleConn(conn, "client"); err != nil {
			t.Error(err)
			errch <- err
		}
	}()

	fmt.Println("test fail", <-errch)
}
