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
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/net/vnode"

	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/net/discovery/protos"
)

/*
150.109.42.150
80.211.128.97
219.75.122.189
94.177.217.223
106.13.0.144
118.25.14.239
39.107.246.16
39.98.64.192
85.112.113.186
51.68.153.155
118.25.182.202
83.166.144.213
80.211.173.196
95.216.216.89
39.100.79.183
80.211.50.31
39.104.100.231
47.94.157.91
150.109.51.8
*/

/*
47.96.31.114
118.25.221.59
47.88.60.116
47.244.133.126
148.70.101.107
59.63.175.4
117.50.66.76
106.75.91.60
35.233.154.15
117.50.22.239
117.50.65.5
117.50.88.123
39.98.58.234
117.50.65.243
*/

func main() {
	target := "119.28.32.48"
	host := target + ":8483"
	targetAddr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		panic(err)
	}

	udpAddr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:8483")
	udp, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	go func() {
		buffer := make([]byte, 1024)
		n, addr, err := udp.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}
		fmt.Printf("receive %d bytes from %s", n, addr)
	}()

	from, err := vnode.EndPoint{
		Host: []byte{127, 0, 0, 1},
		Port: 8483,
		Typ:  vnode.HostIPv4,
	}.Serialize()
	if err != nil {
		panic(err)
	}

	to, err := vnode.ParseEndPoint(host)
	if err != nil {
		panic(err)
	}
	toData, err := to.Serialize()
	if err != nil {
		panic(err)
	}

	ping := &protos.Ping{
		From: from,
		To:   toData,
		Net:  uint32(1),
		Ext:  nil,
		Time: time.Now().Unix(),
	}
	data, err := proto.Marshal(ping)
	if err != nil {
		panic(err)
	}

	_, priv, _ := ed25519.GenerateKey(nil)
	data, _ = pack(priv, 0, data)

	_, err = udp.WriteToUDP(data, targetAddr)
	if err != nil {
		panic(err)
	}

	time.Sleep(5 * time.Second)
}

const version = 0

func pack(priv ed25519.PrivateKey, code byte, payload []byte) (data, hash []byte) {
	hash = crypto.Hash256(payload)

	signature := ed25519.Sign(priv, payload)

	chunk := [][]byte{
		{version, code},
		priv.PubByte(),
		payload,
		signature,
	}

	data = bytes.Join(chunk, nil)

	return
}
