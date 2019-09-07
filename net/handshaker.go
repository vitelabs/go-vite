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
	"time"

	"github.com/vitelabs/go-vite/net/netool"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/net/vnode"
	"github.com/vitelabs/go-vite/vitepb"
)

const handshakeTimeout = 10 * time.Second

type HandshakeMsg struct {
	Version int64

	NetID int64

	Name string

	ID vnode.NodeID

	Timestamp int64

	Height  uint64
	Head    types.Hash
	Genesis types.Hash

	Key   ed25519.PublicKey // is producer
	Token []byte

	FileAddress   []byte
	PublicAddress []byte
}

func (b *HandshakeMsg) Serialize() (data []byte, err error) {
	pb := &vitepb.Handshake{
		Version:       b.Version,
		NetId:         b.NetID,
		Name:          b.Name,
		ID:            b.ID.Bytes(),
		Timestamp:     b.Timestamp,
		Genesis:       b.Genesis.Bytes(),
		Height:        b.Height,
		Head:          b.Head.Bytes(),
		FileAddress:   b.FileAddress,
		Key:           b.Key,
		Token:         b.Token,
		PublicAddress: b.PublicAddress,
	}

	return proto.Marshal(pb)
}

func (b *HandshakeMsg) Deserialize(data []byte) (err error) {
	pb := new(vitepb.Handshake)

	err = proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	b.ID, err = vnode.Bytes2NodeID(pb.ID)
	if err != nil {
		return
	}

	b.Version = pb.Version
	b.NetID = pb.NetId
	b.Name = pb.Name
	b.Timestamp = pb.Timestamp
	b.Height = pb.Height
	b.Head, err = types.BytesToHash(pb.Head)
	if err != nil {
		return
	}
	b.Genesis, err = types.BytesToHash(pb.Genesis)
	if err != nil {
		return
	}
	b.FileAddress = pb.FileAddress
	b.PublicAddress = pb.PublicAddress

	b.Key = pb.Key
	b.Token = pb.Token

	return nil
}

type handshaker struct {
	version       int
	netId         int
	name          string
	id            vnode.NodeID
	genesis       types.Hash
	fileAddress   []byte
	publicAddress []byte

	peerKey ed25519.PrivateKey
	key     ed25519.PrivateKey

	codecFactory CodecFactory

	chain chainReader

	blackList netool.BlackList

	onHandshaker func(c Codec, flag PeerFlag, their *HandshakeMsg) (superior bool, err error)
}

func (h *handshaker) setChain(chain chainReader) {
	h.genesis = chain.GetGenesisSnapshotBlock().Hash
	h.chain = chain
}

func (h *handshaker) banAddr(addr _net.Addr, t int64) {
	addr2, ok := addr.(*_net.TCPAddr)
	var ip _net.IP
	if ok {
		ip = addr2.IP
		h.blackList.Ban(ip, t)
	}
}

func (h *handshaker) bannedAddr(addr _net.Addr) bool {
	addr2, ok := addr.(*_net.TCPAddr)
	var ip _net.IP
	if ok {
		ip = addr2.IP
		return h.blackList.Banned(ip)
	}

	return false
}

func (h *handshaker) readHandshake(c Codec) (their *HandshakeMsg, msgId MsgId, err error) {
	c.SetReadTimeout(handshakeTimeout)
	msg, err := c.ReadMsg()
	if err != nil {
		netLog.Warn(fmt.Sprintf("failed to handshake with %s: read error: %v", c.Address(), err))
		err = PeerNetworkError
		return
	}

	msgId = msg.Id

	if msg.Code == CodeDisconnect {
		if len(msg.Payload) > 0 {
			err = PeerError(msg.Payload[0])
		} else {
			err = PeerUnknownReason
		}
		return
	}

	if msg.Code != CodeHandshake {
		err = PeerNotHandshakeMsg
		netLog.Warn(fmt.Sprintf("failed to handshake with %s: not handshakeMsg %d", c.Address(), msg.Code))
		return
	}

	their = new(HandshakeMsg)
	err = their.Deserialize(msg.Payload)
	if err != nil {
		err = PeerUnmarshalError
		return
	}

	return
}

func (h *handshaker) getSecret(theirId peerId) (secret []byte, err error) {
	pub := ed25519.PublicKey(theirId.Bytes()).ToX25519Pk()
	priv := h.peerKey.ToX25519Sk()

	secret, err = crypto.X25519ComputeSecret(priv, pub)
	if err != nil {
		err = PeerInvalidToken
		return
	}

	return
}

func (h *handshaker) verifyHandshake(their *HandshakeMsg, secret []byte) (err error) {
	t := make([]byte, 8)
	binary.BigEndian.PutUint64(t, uint64(their.Timestamp))
	hash := crypto.Hash256(t)
	token := xor(hash, secret)
	if len(their.Key) != 0 {
		if false == ed25519.Verify(their.Key, token, their.Token) {
			err = PeerInvalidSignature
			return
		}
	} else {
		if false == bytes.Equal(token, their.Token) {
			err = PeerInvalidToken
			return
		}
	}

	return
}

func (h *handshaker) makeHandshake(secret []byte) (our *HandshakeMsg) {
	latestBlock := h.chain.GetLatestSnapshotBlock()
	our = &HandshakeMsg{
		Version:       int64(h.version),
		NetID:         int64(h.netId),
		Name:          h.name,
		ID:            h.id,
		Timestamp:     time.Now().Unix(),
		Height:        latestBlock.Height,
		Head:          latestBlock.Hash,
		Genesis:       h.genesis,
		Key:           nil,
		Token:         nil,
		FileAddress:   h.fileAddress,
		PublicAddress: h.publicAddress,
	}

	t := make([]byte, 8)
	binary.BigEndian.PutUint64(t, uint64(our.Timestamp))
	hash := crypto.Hash256(t)

	our.Token = xor(hash, secret)
	if h.key != nil {
		our.Key = h.key.PubByte()
		our.Token = ed25519.Sign(h.key, our.Token)
	}

	return
}

func (h *handshaker) sendHandshake(c Codec, our *HandshakeMsg, msgId MsgId) (err error) {
	data, err := our.Serialize()
	if err != nil {
		err = PeerUnmarshalError
		return
	}

	c.SetWriteTimeout(handshakeTimeout)
	err = c.WriteMsg(Msg{
		Code:    CodeHandshake,
		Id:      msgId,
		Payload: data,
	})
	if err != nil {
		netLog.Warn(fmt.Sprintf("failed to handshake with %s: write error: %v", c.Address(), err))
		err = PeerNetworkError
		return
	}
	return
}

func (h *handshaker) ReceiveHandshake(conn _net.Conn) (c Codec, their *HandshakeMsg, superior bool, err error) {
	c = h.codecFactory.CreateCodec(conn)

	if h.bannedAddr(conn.RemoteAddr()) {
		err = PeerBanned
		return
	}

	defer func() {
		if err != nil {
			h.banAddr(conn.RemoteAddr(), 60)
		}
	}()

	their, msgId, err := h.readHandshake(c)
	if err != nil {
		return
	}

	secret, err := h.getSecret(their.ID)
	if err != nil {
		return
	}

	err = h.verifyHandshake(their, secret)
	if err != nil {
		return
	}

	err = h.doHandshake(c, PeerFlagInbound, their)
	if err != nil {
		return
	}

	superior, err = h.onHandshaker(c, PeerFlagOutbound, their)
	if err != nil {
		return
	}

	our := h.makeHandshake(secret)
	err = h.sendHandshake(c, our, msgId)
	return
}

func (h *handshaker) InitiateHandshake(conn _net.Conn, id vnode.NodeID) (c Codec, their *HandshakeMsg, superior bool, err error) {
	c = h.codecFactory.CreateCodec(conn)

	secret, err := h.getSecret(id)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			h.blackList.Ban(id.Bytes(), 60)
		}
	}()

	our := h.makeHandshake(secret)
	err = h.sendHandshake(c, our, 0)
	if err != nil {
		return
	}

	their, _, err = h.readHandshake(c)
	if err != nil {
		return
	}

	err = h.verifyHandshake(their, secret)
	if err != nil {
		return
	}

	err = h.doHandshake(c, PeerFlagOutbound, their)
	if err != nil {
		return
	}

	superior, err = h.onHandshaker(c, PeerFlagOutbound, their)
	if err != nil {
		return
	}

	return
}

func (h *handshaker) doHandshake(c Codec, flag PeerFlag, their *HandshakeMsg) (err error) {
	if their.NetID != int64(h.netId) {
		err = PeerDifferentNetwork
		return
	}

	if their.Genesis != h.genesis {
		err = PeerDifferentGenesis
		return
	}

	return
}
