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
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"

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

func (mp *mockProtocol) ReceiveHandshake(msg p2p.HandshakeMsg, protoData []byte, sender net.Addr) (state interface{}, level p2p.Level, err error) {
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

	var port = 8081
	const total = 5
	type sample struct {
		port int
		dir  string
		key  ed25519.PrivateKey
	}
	var samples []sample
	var configs []*p2p.Config
	for i := 0; i < total; i++ {
		var prv ed25519.PrivateKey
		_, prv, err = ed25519.GenerateKey(nil)
		if err != nil {
			panic(err)
		}

		s := sample{
			port: port,
			dir:  ".mock" + strconv.Itoa(i+1),
			key:  prv,
		}
		samples = append(samples, s)

		var cfg *p2p.Config
		cfg, err = p2p.NewConfig("127.0.0.1:"+strconv.Itoa(s.port), "", filepath.Join(pwd, s.dir), prv.Hex(), nil, nil, 10,
			true, s.dir, 0, 0, 0, 0, nil)
		if err != nil {
			panic(err)
		}

		configs = append(configs, cfg)

		port++
	}

	start := func(d p2p.P2P) {
		err = d.Start()
		if err != nil {
			panic(err)
		}
	}

	var ds []p2p.P2P
	for i := 0; i < total; i++ {
		if i != total-1 {
			configs[i].BootNodes = append(configs[i].BootNodes, configs[i+1].Node().ID.String()+"@127.0.0.1:"+strconv.Itoa(samples[i+1].port))
		}

		p := p2p.New(configs[i])
		err = p.Register(&mockProtocol{})
		if err != nil {
			panic(err)
		}
		ds = append(ds, p)
	}

	for _, d := range ds {
		go start(d)
		time.Sleep(time.Second)
	}

	fmt.Println("start")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				for i, d := range ds {
					fmt.Println(i, len(d.Info().Peers))
				}
				fmt.Println("------------")
			}
		}
	}()

	err = http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		panic(err)
	}
}
