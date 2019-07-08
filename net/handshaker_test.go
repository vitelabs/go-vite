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

package net

import (
	"bytes"
	"encoding/binary"
	"fmt"
	_net "net"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/crypto"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/net/netool"
	"github.com/vitelabs/go-vite/net/vnode"
)

func TestProtoHandshakeMsg_Serialize(t *testing.T) {
	msg := HandshakeMsg{
		Version:       12,
		NetID:         76,
		Name:          "vite",
		ID:            vnode.RandomNodeID(),
		Timestamp:     time.Now().Unix() - 100,
		Height:        9384,
		Head:          types.Hash{1, 2, 4},
		Genesis:       types.Hash{4, 5, 6},
		Key:           []byte{1, 2, 3},
		Token:         []byte{5, 6, 7},
		FileAddress:   []byte{1, 2},
		PublicAddress: []byte{3, 4},
	}

	data, err := msg.Serialize()
	if err != nil {
		t.Error("serialize error")
	}

	var msg2 = &HandshakeMsg{}
	err = msg2.Deserialize(data)
	if err != nil {
		t.Errorf("deserialize error")
	}

	if msg.Version != msg2.Version {
		t.Errorf("different version: %d %d", msg.Version, msg2.Version)
	}
	if msg.NetID != msg2.NetID {
		t.Errorf("different net: %d %d", msg.NetID, msg2.NetID)
	}
	if msg.Name != msg2.Name {
		t.Errorf("different name: %s %s", msg.Name, msg2.Name)
	}
	if msg.ID != msg2.ID {
		t.Errorf("different id")
	}
	if msg.Timestamp != msg2.Timestamp {
		t.Errorf("different time: %d %d", msg.Timestamp, msg2.Timestamp)
	}
	if msg.Height != msg2.Height {
		t.Errorf("different height: %d %d", msg.Height, msg2.Height)
	}
	if msg.Head != msg2.Head {
		t.Errorf("different head: %s %s", msg.Head, msg2.Head)
	}
	if msg.Genesis != msg2.Genesis {
		t.Errorf("different genesis: %s %s", msg.Genesis, msg2.Genesis)
	}
	if false == bytes.Equal(msg.Key, msg2.Key) {
		t.Errorf("different key: %v %v", msg.Key, msg2.Key)
	}
	if false == bytes.Equal(msg.FileAddress, msg2.FileAddress) {
		t.Errorf("different fileAddress: %v %v", msg.FileAddress, msg2.FileAddress)
	}
	if false == bytes.Equal(msg.PublicAddress, msg2.PublicAddress) {
		t.Errorf("different publicAddress: %v %v", msg.PublicAddress, msg2.PublicAddress)
	}
	if false == bytes.Equal(msg.Token, msg2.Token) {
		t.Errorf("different token: %v %v", msg.Token, msg2.Token)
	}
}

func TestExtractFileAddress(t *testing.T) {
	type sample struct {
		sender *_net.TCPAddr
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
			sender: &_net.TCPAddr{
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
			sender: &_net.TCPAddr{
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
		{
			sender: &_net.TCPAddr{
				IP:   []byte{117, 0, 0, 3},
				Port: 8483,
			},
			buf: buf,
			handle: func(str string) error {
				if str != "117.0.0.3:8484" {
					return fmt.Errorf("wrong fileAddress: %s", str)
				}
				return nil
			},
		},
	}

	for _, samp := range samples {
		if err = samp.handle(extractAddress(samp.sender, samp.buf, 8484)); err != nil {
			t.Error(err)
		}
	}
}

func TestHandshake(t *testing.T) {
	pub1, priv1, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	_, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	pub3, priv3, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	_, priv4, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	id1, _ := vnode.Bytes2NodeID(pub1)
	id2, _ := vnode.Bytes2NodeID(pub3)

	codecFac := &transportFactory{
		minCompressLength: 100,
		readTimeout:       readMsgTimeout,
		writeTimeout:      writeMsgTimeout,
	}

	var wg sync.WaitGroup

	const addr = "127.0.0.1:9999"
	wg.Add(1)
	go func() {
		defer wg.Done()

		hk := &handshaker{
			version:       1,
			netId:         7,
			name:          "node1",
			id:            id1,
			fileAddress:   nil,
			publicAddress: nil,
			peerKey:       priv1,
			key:           priv2,
			codecFactory:  codecFac,
			chain:         nil,
			blackList: netool.NewBlackList(func(t int64, count int) bool {
				return false
			}),
			onHandshaker: func(c Codec, flag PeerFlag, their *HandshakeMsg) (superior bool, err error) {
				fmt.Printf("%+v\n", their)
				return false, nil
			},
		}
		hk.setChain(mockChain{
			height: 100,
		})

		ln, err := _net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		_, _, _, err = hk.ReceiveHandshake(conn)
		if err != nil {
			panic(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var e = vnode.EndPoint{
			Host: []byte{117, 0, 0, 1},
			Port: 8888,
			Typ:  vnode.HostIPv4,
		}
		publicAddress, _ := e.Serialize()
		e = vnode.EndPoint{
			Host: []byte{117, 0, 0, 2},
			Port: 9999,
			Typ:  vnode.HostIPv4,
		}
		fileAddress, _ := e.Serialize()

		hk := &handshaker{
			version:       1,
			netId:         7,
			name:          "node2",
			id:            id2,
			fileAddress:   fileAddress,
			publicAddress: publicAddress,
			peerKey:       priv3,
			key:           priv4,
			codecFactory:  codecFac,
			chain:         nil,
			blackList: netool.NewBlackList(func(t int64, count int) bool {
				return false
			}),
			onHandshaker: func(c Codec, flag PeerFlag, their *HandshakeMsg) (superior bool, err error) {
				fmt.Printf("%+v\n", their)
				return false, nil
			},
		}
		hk.setChain(mockChain{
			height: 200,
		})

		time.Sleep(time.Second)
		conn, err := _net.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}

		_, _, _, err = hk.InitiateHandshake(conn, id1)
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}

func TestHandshaker_makeHandshake(t *testing.T) {
	_, priv1, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	_, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	_, priv3, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	once := false
check:

	hkr := &handshaker{
		peerKey: priv1,
		key:     priv3,
	}
	hkr.setChain(mockChain{
		height: 111,
	})
	hkr.id, _ = vnode.Bytes2NodeID(priv1.PubByte())

	theirId, _ := vnode.Bytes2NodeID(priv2.PubByte())
	secret, err := hkr.getSecret(theirId)
	if err != nil {
		panic(err)
	}

	our := hkr.makeHandshake(secret)
	err = hkr.verifyHandshake(our, secret)
	if err != nil {
		panic(err)
	}

	hkr.key = nil
	if !once {
		once = true
		goto check
	}
}

func TestHandshaker_verifyHandshake(t *testing.T) {
	pub1, priv1, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	_, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	_, priv3, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	_, priv4, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	hkr := &handshaker{
		peerKey: priv1,
		key:     priv3,
	}
	hkr.setChain(mockChain{
		height: 111,
	})
	hkr.id, _ = vnode.Bytes2NodeID(pub1)

	makeHandshake := func(peerKey, mineKey ed25519.PrivateKey, theirId vnode.NodeID) (secret []byte, our *HandshakeMsg) {
		ourId, _ := vnode.Bytes2NodeID(peerKey.PubByte())
		our = &HandshakeMsg{
			Version:       2,
			NetID:         3,
			Name:          "hello",
			ID:            ourId,
			Timestamp:     1000,
			Height:        0,
			Head:          types.Hash{},
			Genesis:       types.Hash{},
			FileAddress:   nil,
			PublicAddress: nil,
		}

		pub := ed25519.PublicKey(theirId.Bytes()).ToX25519Pk()
		priv := peerKey.ToX25519Sk()
		secret, err := crypto.X25519ComputeSecret(priv, pub)
		if err != nil {
			panic(err)
		}

		t := make([]byte, 8)
		binary.BigEndian.PutUint64(t, uint64(our.Timestamp))
		hash := crypto.Hash256(t)
		our.Token = xor(hash, secret)
		if mineKey != nil {
			our.Key = mineKey.PubByte()
			our.Token = ed25519.Sign(mineKey, our.Token)
		}

		return
	}

	secret, our := makeHandshake(priv2, priv4, hkr.id)
	err = hkr.verifyHandshake(our, secret)
	if err != nil {
		panic(err)
	}

	our.Timestamp = time.Now().Unix()
	err = hkr.verifyHandshake(our, secret)
	if err == nil {
		t.Fatal("should verify failed")
	} else {
		fmt.Println(err)
	}

	_, our = makeHandshake(priv2, nil, hkr.id)
	err = hkr.verifyHandshake(our, secret)
	if err != nil {
		panic(err)
	}
}
