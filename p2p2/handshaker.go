package p2p

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/p2p2/protos"

	"github.com/vitelabs/go-vite/p2p2/vnode"

	"github.com/vitelabs/go-vite/log15"
)

const (
	baseProtocolID = 0
	baseHandshake  = 1
	baseDisconnect = 2
	baseTooManyMsg = 3
)

const handshakeTimeout = 3 * time.Second

const nonceLen = 32

var (
	errNotBaseProtocolMsg = errors.New("not base protocol message")
	errNotHandshakeMsg    = errors.New("not handshake message")
	errInvalidSign        = errors.New("invalid signature")
	errReadTooShort       = errors.New("read too short")
	errDifferentNetwork   = errors.New("different network")
)

type CodecFactory = func(conn net.Conn) Codec

type protoData struct {
	id   ProtocolID
	data []byte
}

type protoDataList []protoData

func (pd protoDataList) Len() int {
	return len(pd)
}

func (pd protoDataList) Less(i, j int) bool {
	return pd[i].id < pd[j].id
}

func (pd protoDataList) Swap(i, j int) {
	pd[i], pd[j] = pd[j], pd[i]
}

type baseHandshakeMsg struct {
	version uint16

	netId uint16

	name string

	id vnode.NodeID

	timestamp int64

	nonce []byte
}

func (b *baseHandshakeMsg) Serialize() (data []byte, err error) {
	pb := &protos.BaseHandshakeMsg{
		Version:   uint32(b.version),
		NetId:     uint32(b.netId),
		Name:      b.name,
		ID:        b.id[:],
		Timestamp: b.timestamp,
		Nonce:     b.nonce,
	}

	return proto.Marshal(pb)
}

type handshakeMsg struct {
	baseHandshakeMsg

	protoData []*protos.ProtoData
}

type defaultHandshaker struct {
	version uint16
	netId   uint16
	name    string
	priv    ed25519.PrivateKey

	fac     CodecFactory
	ptMap   ProtocolMap
	netTool netutils

	log log15.Logger
}

func (h *defaultHandshaker) versionShake(conn net.Conn) (c Codec, version uint16, err error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[:2], h.version)
	binary.BigEndian.PutUint16(buf[2:], h.netId)

	_ = conn.SetWriteDeadline(time.Now().Add(handshakeTimeout))
	n, err := conn.Write(buf)
	if err != nil {
		return
	}
	if n != len(buf) {
		err = errWriteTooShort
		return
	}

	_ = conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	n, err = io.ReadFull(conn, buf)
	if err != nil {
		return
	}
	if n != len(buf) {
		err = errReadTooShort
		return
	}

	version, netId := binary.BigEndian.Uint16(buf[:2]), binary.BigEndian.Uint16(buf[2:])

	if netId != h.netId {
		err = errDifferentNetwork
		h.log.Error(fmt.Sprintf("peer %s is from different network %d, our %d", conn.RemoteAddr(), netId, h.netId))
		return
	}

	c = h.fac(conn)
	if version != h.version {
		_ = c.WriteMsg(Msg{
			Pid:     baseProtocolID,
			Code:    baseDisconnect,
			Payload: PeerIncompatibleVersion.Serialize(),
		})

		err = PeerIncompatibleVersion

		h.log.Error(fmt.Sprintf("peer %s version %d, our %d", conn.RemoteAddr(), version, h.version))
		return
	}

	return
}

func (h *defaultHandshaker) initiateHandshake(conn net.Conn) (peer Peer, err error) {
	codec, version, err := h.versionShake(conn)
	if err != nil {
		return
	}

	// todo handshake data
	var id vnode.NodeID
	copy(id[:], h.priv.PubByte())

	nonce := make([]byte, nonceLen)
	_, _ = io.ReadFull(crand.Reader, nonce)

	hsm := handshakeMsg{
		baseHandshakeMsg: baseHandshakeMsg{
			version:   h.version,
			netId:     h.netId,
			name:      h.name,
			id:        id,
			timestamp: time.Now().Unix(),
			nonce:     nonce,
		},
	}

	// base protocol data
	baseData, err := hsm.baseHandshakeMsg.Serialize()
	if err != nil {
		return
	}

	// add protocol data
	for pid, pt := range h.ptMap {
		hsm.protoData = append(hsm.protoData, &protos.ProtoData{
			ID:   uint32(pid),
			Data: pt.Auth(baseData),
		})
	}

	hsmpb := &protos.HandshakeMsg{
		Base:      baseData,
		Protocols: hsm.protoData,
	}
	data, err := proto.Marshal(hsmpb)
	if err != nil {
		return
	}

	codec.SetWriteTimeout(handshakeTimeout)
	err = codec.WriteMsg(Msg{
		Pid:     0,
		Code:    baseHandshake,
		Payload: nil,
	})

	if err != nil {
		return
	}

	codec.SetReadTimeout(handshakeTimeout)
	msg, err := codec.ReadMsg()
	if err != nil {
		return
	}

	if msg.Pid != baseProtocolID {
		err = errNotBaseProtocolMsg
		return
	}

	if msg.Code == baseDisconnect {
		err = deserialize(msg.Payload)
		return
	}

	if msg.Code != baseHandshake {
		err = errNotHandshakeMsg
		err = codec.WriteMsg(Msg{
			Pid:     0,
			Code:    baseDisconnect,
			Payload: nil,
		})
		return
	}

	// todo create peer
	return
}

func (h *defaultHandshaker) receiveHandshake(conn net.Conn) (peer Peer, err error) {
	codec, version, err := h.versionShake(conn)
	if err != nil {
		return
	}

	codec.SetReadTimeout(5 * time.Second)
	msg, err := codec.ReadMsg()
	if err != nil {
		return
	}

	if msg.Pid != baseProtocolID {
		err = errNotBaseProtocolMsg
		return
	}

	if msg.Code == baseDisconnect {
		err = deserialize(msg.Payload)
		return
	}

	if msg.Code != baseHandshake {
		err = errNotHandshakeMsg
		err = codec.WriteMsg(Msg{
			Pid:     0,
			Code:    baseDisconnect,
			Payload: nil,
		})
		return
	}

	// todo create peer

	return
}

func newHandshaker(fac CodecFactory, ptMap map[ProtocolID]Protocol, netTool netutils, log log15.Logger) handshaker {
	return &defaultHandshaker{
		fac:     fac,
		ptMap:   ptMap,
		netTool: netTool,
		log:     log,
	}
}
