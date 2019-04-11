package net

import (
	"fmt"
	net2 "net"
	"testing"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestExtractFileAddress(t *testing.T) {
	type sample struct {
		sender net2.Addr
		buf    []byte
		handle func(str string) error
	}

	var e = vnode.EndPoint{
		Host: []byte("localhost"),
		Port: 8484,
		Typ:  vnode.HostDomain,
	}
	buf, err := e.Serialize()
	if err != nil {
		panic(err)
	}

	var samples = []sample{
		{
			sender: &net2.TCPAddr{
				IP:   []byte{192, 168, 0, 1},
				Port: 8483,
			},
			buf: []byte{33, 36},
			handle: func(str string) error {
				if str != "192.168.0.1:8484" {
					return fmt.Errorf("wrong fileAddress: %s", str)
				}
				return nil
			},
		},
		{
			sender: &net2.TCPAddr{
				IP:   []byte{192, 168, 0, 1},
				Port: 8483,
			},
			buf: buf,
			handle: func(str string) error {
				if str != "192.168.0.1:8484" {
					return fmt.Errorf("wrong fileAddress: %s", str)
				}
				return nil
			},
		},
	}

	for _, samp := range samples {
		if err := samp.handle(extractFileAddress(samp.sender, samp.buf)); err != nil {
			t.Error(err)
		}
	}
}
