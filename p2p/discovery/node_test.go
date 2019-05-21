package discovery

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestExtractEndPoint(t *testing.T) {
	type sample struct {
		sender *net.UDPAddr
		from   string
		handle func(e *vnode.EndPoint, addr *net.UDPAddr) error
	}
	samples := []sample{
		{
			sender: &net.UDPAddr{
				IP:   []byte{127, 0, 0, 1},
				Port: 8483,
			},
			from: "vite.org",
			handle: func(e *vnode.EndPoint, addr *net.UDPAddr) error {
				if e.Typ.Is(vnode.HostIP) {
					return errors.New("should be domain")
				}

				if e.String() != "vite.org:8483" {
					return fmt.Errorf("error endpoint: %s", e.String())
				}
				return nil
			},
		},
		{
			sender: &net.UDPAddr{
				IP:   []byte{193, 110, 91, 250},
				Port: 8483,
			},
			from: "0.0.0.0:8483",
			handle: func(e *vnode.EndPoint, addr *net.UDPAddr) error {
				if !e.Typ.Is(vnode.HostIPv4) {
					return errors.New("should be ipv4")
				}

				if e.String() != "193.110.91.250:8483" {
					return fmt.Errorf("error endpoint: %s", e.String())
				}
				return nil
			},
		},
		{
			sender: &net.UDPAddr{
				IP:   []byte{193, 110, 91, 250},
				Port: 8483,
			},
			from: "127.0.0.1:8483",
			handle: func(e *vnode.EndPoint, addr *net.UDPAddr) error {
				if !e.Typ.Is(vnode.HostIPv4) {
					return errors.New("should be ipv4")
				}

				if e.String() != "193.110.91.250:8483" {
					return fmt.Errorf("error endpoint: %s", e.String())
				}
				return nil
			},
		},
		{
			sender: &net.UDPAddr{
				IP:   []byte{192, 168, 1, 36},
				Port: 8483,
			},
			from: "127.0.0.1:8483",
			handle: func(e *vnode.EndPoint, addr *net.UDPAddr) error {
				if !e.Typ.Is(vnode.HostIPv4) {
					return errors.New("should be ipv4")
				}

				if e.String() != "192.168.1.36:8483" {
					return fmt.Errorf("error endpoint: %s", e.String())
				}
				return nil
			},
		},
		{
			sender: &net.UDPAddr{
				IP:   []byte{192, 168, 1, 36},
				Port: 8483,
			},
			from: "0.0.0.0:8888",
			handle: func(e *vnode.EndPoint, addr *net.UDPAddr) error {
				if !e.Typ.Is(vnode.HostIPv4) {
					return errors.New("should be ipv4")
				}

				if e.String() != "192.168.1.36:8483" {
					return fmt.Errorf("error endpoint: %s", e.String())
				}
				return nil
			},
		},
	}

	for _, samp := range samples {
		from, err := vnode.ParseEndPoint(samp.from)
		if err != nil {
			panic(err)
		}

		e, addr := extractEndPoint(samp.sender, &from)

		if err = samp.handle(e, addr); err != nil {
			t.Error(err)
		}
	}
}
