/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/vitelabs/go-vite/p2p"
)

type mockProtocol struct {
}

func (mp *mockProtocol) Name() string {
	return "mock protocol"
}

func (mp *mockProtocol) ID() p2p.ProtocolID {
	return 2
}

func (mp *mockProtocol) ProtoData() []byte {
	return nil
}

func (mp *mockProtocol) ReceiveHandshake(msg p2p.HandshakeMsg, protoData []byte) (state interface{}, level p2p.Level, err error) {
	return
}

func (mp *mockProtocol) Handle(msg p2p.Msg) error {
	fmt.Printf("receive message %d from %s\n", msg.Code, msg.Sender.Address())
	return nil
}

func (mp *mockProtocol) State() []byte {
	return nil
}

func (mp *mockProtocol) SetState(state []byte, peer p2p.Peer) {
	return
}

func (mp *mockProtocol) OnPeerAdded(peer p2p.Peer) error {
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			err := peer.WriteMsg(p2p.Msg{
				Code:    0,
				Id:      0,
				Payload: []byte("hello"),
			})
			if err != nil {
				_ = peer.Close(p2p.PeerError(p2p.PeerInvalidSignature))
			}
		}
	}()

	return nil
}

func (mp *mockProtocol) OnPeerRemoved(peer p2p.Peer) error {
	return nil
}

func main() {
	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	cfg1, err := p2p.NewConfig("127.0.0.1:8483", "127.0.0.1:8483", filepath.Join(pwd, ".mock1"), "", nil, nil, 10,
		true, "mock1", 0, 0, 0, 0, nil)
	if err != nil {
		panic(err)
	}

	cfg2, err := p2p.NewConfig("127.0.0.1:8484", "127.0.0.1:8484", filepath.Join(pwd, ".mock2"), "", nil, nil, 10,
		true, "mock2", 0, 0, 0, 0, nil)
	if err != nil {
		panic(err)
	}

	cfg3, err := p2p.NewConfig("127.0.0.1:8485", "127.0.0.1:8485", filepath.Join(pwd, ".mock3"), "", nil, nil, 10,
		true, "mock3", 0, 0, 0, 0, nil)
	if err != nil {
		panic(err)
	}

	cfg4, err := p2p.NewConfig("127.0.0.1:8486", "127.0.0.1:8486", filepath.Join(pwd, ".mock4"), "", nil, nil, 10,
		true, "mock4", 0, 0, 0, 0, nil)
	if err != nil {
		panic(err)
	}

	cfg5, err := p2p.NewConfig("127.0.0.1:8487", "127.0.0.1:8487", filepath.Join(pwd, ".mock5"), "", nil, nil, 10,
		true, "mock3", 0, 0, 0, 0, nil)
	if err != nil {
		panic(err)
	}

	cfg1.BootNodes = append(cfg1.BootNodes, cfg2.Node().String())
	cfg2.BootNodes = append(cfg2.BootNodes, cfg3.Node().String())
	cfg3.BootNodes = append(cfg3.BootNodes, cfg4.Node().String())
	cfg4.BootNodes = append(cfg4.BootNodes, cfg5.Node().String())

	d1 := p2p.New(cfg1)
	d2 := p2p.New(cfg2)
	d3 := p2p.New(cfg3)
	d4 := p2p.New(cfg4)
	d5 := p2p.New(cfg5)

	err = d1.Register(&mockProtocol{})
	if err != nil {
		panic(err)
	}
	err = d2.Register(&mockProtocol{})
	if err != nil {
		panic(err)
	}
	err = d3.Register(&mockProtocol{})
	if err != nil {
		panic(err)
	}
	err = d4.Register(&mockProtocol{})
	if err != nil {
		panic(err)
	}
	err = d5.Register(&mockProtocol{})
	if err != nil {
		panic(err)
	}

	start := func(p p2p.P2P) {
		err = p.Start()
		if err != nil {
			panic(err)
		}
	}

	go start(d1)
	time.Sleep(time.Second)
	go start(d2)
	time.Sleep(time.Second)
	go start(d3)
	time.Sleep(time.Second)
	go start(d4)
	time.Sleep(time.Second)
	go start(d5)

	fmt.Println("start")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				fmt.Println("d1", d1.Info())
				fmt.Println("d2", d2.Info())
				fmt.Println("d3", d3.Info())
				fmt.Println("d4", d4.Info())
				fmt.Println("d5", d5.Info())
				fmt.Println("------------")
			}
		}
	}()

	err = http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		panic(err)
	}
}
