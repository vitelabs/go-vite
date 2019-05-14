package p2p

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestHandshakeMsg_Serialize(t *testing.T) {
	var msg HandshakeMsg
	data, err := msg.Serialize()
	if err != nil {
		t.Error("serialize error")
	}

	var msg2 HandshakeMsg
	err = msg2.Deserialize(data)
	if err != nil {
		t.Error("deserialize error")
	}
}

func TestProtoHandshakeMsg_Serialize(t *testing.T) {
	msg := HandshakeMsg{
		Version:     1,
		NetID:       7,
		Name:        "vite",
		ID:          vnode.RandomNodeID(),
		Timestamp:   time.Now().Unix() - 100,
		Height:      9384,
		Head:        types.Hash{1, 2, 4},
		Genesis:     types.Hash{4, 5, 6},
		Key:         []byte{1, 2, 3},
		Token:       []byte{5, 6, 7},
		FileAddress: []byte{1, 2},
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
	if false == bytes.Equal(msg.Token, msg2.Token) {
		t.Errorf("different token: %v %v", msg.Token, msg2.Token)
	}

}

func TestExtractFileAddress(t *testing.T) {
	type sample struct {
		sender net.Addr
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
			sender: &net.TCPAddr{
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
			sender: &net.TCPAddr{
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

	var genesis = types.Hash{1, 2, 3}

	var wg sync.WaitGroup

	const addr = "127.0.0.1:9999"
	wg.Add(1)
	go func() {
		defer wg.Done()

		hk := &handshaker{
			version:     1,
			netId:       7,
			name:        "node1",
			id:          id1,
			genesis:     genesis,
			fileAddress: nil,
			peerKey:     priv1,
			key:         priv2,
			protocol:    &mockProtocol{},
		}

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		c := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		_, err = hk.ReceiveHandshake(c)
		if err != nil {
			panic(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		hk := &handshaker{
			version:  1,
			netId:    7,
			name:     "node2",
			id:       id2,
			genesis:  genesis,
			peerKey:  priv3,
			key:      priv4,
			protocol: &mockProtocol{},
		}

		time.Sleep(time.Second)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}
		c := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)
		_, err = hk.InitiateHandshake(c, id1)
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}
