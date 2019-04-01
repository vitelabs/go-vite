package p2p

import (
	"net"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/p2p/protos"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/log15"
)

const (
	baseProtocolID  = 0
	baseEncRequest  = 1
	baseEncResponse = 2
	baseHandshake   = 3
	baseDisconnect  = 4 // body is struct Error
	baseTooManyMsg  = 5
	baseHeartBeat   = 6
)

const version = iota
const handshakeTimeout = 3 * time.Second

const nonceLen = 32
const signatureLen = 64

type protoDataList []*protos.Protocol

func (pd protoDataList) Len() int {
	return len(pd)
}

func (pd protoDataList) Less(i, j int) bool {
	return pd[i].ID < pd[j].ID
}

func (pd protoDataList) Swap(i, j int) {
	pd[i], pd[j] = pd[j], pd[i]
}

type HandshakeMsg struct {
	Version uint32

	NetID uint32

	Name string

	ID vnode.NodeID

	Timestamp int64

	protocols protoDataList
}

func (b *HandshakeMsg) Serialize() (data []byte, err error) {
	pb := &protos.Handshake{
		Version:   b.Version,
		NetId:     b.NetID,
		Name:      b.Name,
		ID:        b.ID.Bytes(),
		Timestamp: b.Timestamp,
		Protocols: b.protocols,
	}

	return proto.Marshal(pb)
}

func (b *HandshakeMsg) Deserialize(data []byte) (err error) {
	pb := new(protos.Handshake)

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
	b.protocols = pb.Protocols

	return nil
}

type handshaker struct {
	version uint32
	netId   uint32
	name    string
	id      vnode.NodeID

	priv ed25519.PrivateKey

	codecFactory CodecFactory
	ptMap        ProtocolMap

	log log15.Logger
}

func (h *handshaker) catch(codec Codec, err *Error) {
	_ = Disconnect(codec, err)
	_ = codec.Close()
}

func (h *handshaker) sendHandshake(codec Codec) (err error) {
	hsm := HandshakeMsg{
		Version:   h.version,
		NetID:     h.netId,
		Name:      h.name,
		ID:        h.id,
		Timestamp: time.Now().Unix(),
		protocols: nil,
	}
	for id, pt := range h.ptMap {
		hsm.protocols = append(hsm.protocols, &protos.Protocol{
			ID:   uint32(id),
			Data: pt.ProtoData(),
		})
	}

	hspkt, err := hsm.Serialize()
	if err != nil {
		return
	}

	// add signature to head
	sign := ed25519.Sign(h.priv, hspkt)
	hspkt = append(sign, hspkt...)

	codec.SetWriteTimeout(handshakeTimeout)
	err = codec.WriteMsg(Msg{
		Pid:     0,
		Code:    baseHandshake,
		Payload: hspkt,
	})

	if err != nil {
		return
	}

	return nil
}

func (h *handshaker) readHandshake(codec Codec) (their *HandshakeMsg, err error) {
	codec.SetReadTimeout(handshakeTimeout)
	msg, err := codec.ReadMsg()
	if err != nil {
		return nil, PeerNetworkError
	}

	if msg.Pid != baseProtocolID {
		return nil, PeerNotHandshakeMsg
	}

	if msg.Code == baseDisconnect {
		var e = new(Error)
		err = e.Deserialize(msg.Payload)
		if err != nil {
			return nil, err
		}

		return nil, e
	}

	if msg.Code != baseHandshake {
		return nil, PeerNotHandshakeMsg
	}

	if len(msg.Payload) < signatureLen {
		return nil, PeerPayloadTooShort
	}

	sign := msg.Payload[:signatureLen]
	their = new(HandshakeMsg)
	err = their.Deserialize(msg.Payload[signatureLen:])
	if err != nil {
		return nil, PeerUnmarshalError
	}

	ok := ed25519.Verify(their.ID.Bytes(), msg.Payload[signatureLen:], sign)
	if !ok {
		return nil, PeerInvalidSignature
	}

	return
}

func (h *handshaker) doHandshake(codec Codec, level Level) (their *HandshakeMsg, ptMap map[ProtocolID]peerProtocol, level2 Level, err error) {
	err = h.sendHandshake(codec)
	if err != nil {
		return
	}

	their, err = h.readHandshake(codec)
	if err != nil {
		return
	}

	if their.NetID != h.netId {
		err = PeerDifferentNetwork
		return
	}

	if their.Version < h.version {
		err = PeerIncompatibleVersion
		return
	}

	// protocols
	ptMap, level2, err = matchProtocols(h.ptMap, their, level)
	if err != nil {
		err = &Error{
			Code:    uint32(PeerProtocolError),
			Message: err.Error(),
		}

		return
	}

	if len(ptMap) == 0 {
		err = PeerNoMatchedProtocols
		return
	}

	return
}

func (h *handshaker) Handshake(conn net.Conn, level Level) (peer PeerMux, err error) {
	codec := h.codecFactory.CreateCodec(conn)

	their, ptMap, level, err := h.doHandshake(codec, level)

	if err != nil {
		var e *Error
		var ok bool
		var pe PeerError
		if pe, ok = err.(PeerError); ok {
			e = &Error{
				Code: uint32(pe),
			}
		} else if e, ok = err.(*Error); ok {
			// do nothing
		} else {
			e = &Error{
				Message: err.Error(),
			}
		}

		h.catch(codec, e)
		return
	}

	peer = NewPeer(their.ID, their.Name, their.Version, codec, level, ptMap)

	return
}

func matchProtocols(our ProtocolMap, their *HandshakeMsg, level1 Level) (ptMap map[ProtocolID]peerProtocol, level2 Level, err error) {
	ptMap = make(map[ProtocolID]peerProtocol)

	level2 = level1

	var ptme Protocol
	var plevel Level
	var ok bool
	var state interface{}
	for _, pt := range their.protocols {
		pid := ProtocolID(pt.ID)

		if ptme, ok = our[pid]; ok {
			if state, plevel, err = ptme.ReceiveHandshake(*their, pt.Data); err != nil {
				return nil, level1, err
			} else {
				ptMap[pid] = peerProtocol{
					Protocol: ptme,
					state:    state,
				}

				if plevel > level1 {
					level2 = plevel
				}
			}
		}
	}

	return
}
