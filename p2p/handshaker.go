package p2p

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/crypto"

	"github.com/vitelabs/go-vite/vitepb"

	"github.com/vitelabs/go-vite/p2p/netool"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/log15"
)

const (
	CodeDisconnect  Code = 1
	CodeHandshake   Code = 2
	CodeControlFlow Code = 3
	CodeHeartBeat   Code = 4

	CodeGetHashList       Code = 25
	CodeHashList          Code = 26
	CodeGetSnapshotBlocks Code = 27
	CodeSnapshotBlocks    Code = 28
	CodeGetAccountBlocks  Code = 29
	CodeAccountBlocks     Code = 30
	CodeNewSnapshotBlock  Code = 31
	CodeNewAccountBlock   Code = 32

	CodeSyncHandshake   Code = 60
	CodeSyncHandshakeOK Code = 61
	CodeSyncRequest     Code = 62
	CodeSyncReady       Code = 63

	CodeException Code = 127
	CodeTrace     Code = 128
)

const version = iota
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

	FileAddress []byte
}

func (b *HandshakeMsg) Serialize() (data []byte, err error) {
	pb := &vitepb.Handshake{
		Version:     b.Version,
		NetId:       b.NetID,
		Name:        b.Name,
		ID:          b.ID.Bytes(),
		Timestamp:   b.Timestamp,
		Genesis:     b.Genesis.Bytes(),
		Height:      b.Height,
		Head:        b.Head.Bytes(),
		FileAddress: b.FileAddress,
		Key:         b.Key,
		Token:       b.Token,
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

	b.Key = pb.Key
	b.Token = pb.Token

	return nil
}

type handshaker struct {
	version     int
	netId       int
	name        string
	id          vnode.NodeID
	genesis     types.Hash
	fileAddress []byte

	peerKey ed25519.PrivateKey
	key     ed25519.PrivateKey

	protocol Protocol

	log log15.Logger
}

func (h *handshaker) readHandshake(c Codec) (their *HandshakeMsg, id MsgId, err error) {
	c.SetReadTimeout(handshakeTimeout)
	msg, err := c.ReadMsg()
	if err != nil {
		err = PeerNetworkError
		return
	}

	id = msg.Id

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

func (h *handshaker) ReceiveHandshake(c Codec) (peer PeerMux, err error) {
	their, msgId, err := h.readHandshake(c)
	if err != nil {
		return
	}

	pub := ed25519.PublicKey(their.ID.Bytes()).ToX25519Pk()
	priv := h.peerKey.ToX25519Sk()
	secret, err := crypto.X25519ComputeSecret(priv, pub)
	if err != nil {
		return nil, PeerInvalidToken
	}
	t := make([]byte, 8)
	binary.BigEndian.PutUint64(t, uint64(their.Timestamp))
	hash := crypto.Hash256(t)
	token := xor(hash, secret)
	if len(their.Key) != 0 {
		if false == ed25519.Verify(their.Key, token, their.Token) {
			return nil, PeerInvalidSignature
		}
	} else {
		if false == bytes.Equal(token, their.Token) {
			return nil, PeerInvalidToken
		}
	}

	peer, err = h.doHandshake(c, Inbound, their)
	if err != nil {
		return nil, err
	}

	response := HandshakeMsg{
		Version:     int64(h.version),
		NetID:       int64(h.netId),
		Name:        h.name,
		ID:          h.id,
		Timestamp:   time.Now().Unix(),
		FileAddress: h.fileAddress,
	}
	response.Height, response.Head, response.Genesis = h.protocol.ProtoData()
	binary.BigEndian.PutUint64(t, uint64(response.Timestamp))
	hash = crypto.Hash256(t)
	response.Token = xor(hash, secret)
	if len(h.key) != 0 {
		response.Key = h.key.PubByte()
		response.Token = ed25519.Sign(h.key, response.Token)
	}
	data, err := response.Serialize()
	if err != nil {
		return nil, PeerUnmarshalError
	}

	c.SetWriteTimeout(handshakeTimeout)
	err = c.WriteMsg(Msg{
		Code:    CodeHandshake,
		Id:      msgId,
		Payload: data,
	})
	if err != nil {
		return nil, PeerNetworkError
	}

	return
}

func (h *handshaker) InitiateHandshake(c Codec, id vnode.NodeID) (peer PeerMux, err error) {
	request := HandshakeMsg{
		Version:     int64(h.version),
		NetID:       int64(h.netId),
		Name:        h.name,
		ID:          h.id,
		Timestamp:   time.Now().Unix(),
		FileAddress: h.fileAddress,
	}
	request.Height, request.Head, request.Genesis = h.protocol.ProtoData()

	t := make([]byte, 8)
	binary.BigEndian.PutUint64(t, uint64(request.Timestamp))
	hash := crypto.Hash256(t)
	priv := h.peerKey.ToX25519Sk()
	pub := ed25519.PublicKey(id.Bytes()).ToX25519Pk()
	secret, err := crypto.X25519ComputeSecret(priv, pub)
	if err != nil {
		return nil, PeerInvalidToken
	}

	request.Token = xor(hash, secret)
	if h.key != nil {
		request.Key = h.key.PubByte()
		request.Token = ed25519.Sign(h.key, request.Token)
	}
	data, err := request.Serialize()
	if err != nil {
		return nil, PeerUnmarshalError
	}

	c.SetWriteTimeout(handshakeTimeout)
	err = c.WriteMsg(Msg{
		Code:    CodeHandshake,
		Payload: data,
	})
	if err != nil {
		return nil, PeerNetworkError
	}

	their, _, err := h.readHandshake(c)
	if err != nil {
		return
	}

	binary.BigEndian.PutUint64(t, uint64(their.Timestamp))
	hash = crypto.Hash256(t)
	token := xor(hash, secret)
	if len(their.Key) != 0 {
		if false == ed25519.Verify(their.Key, token, their.Token) {
			return nil, PeerInvalidSignature
		}
	} else if false == bytes.Equal(their.Token, token) {
		return nil, PeerInvalidToken
	}

	return h.doHandshake(c, Outbound, their)
}

func (h *handshaker) doHandshake(c Codec, level Level, their *HandshakeMsg) (peer PeerMux, err error) {
	if their.NetID != int64(h.netId) {
		err = PeerDifferentNetwork
		return
	}

	if their.Version < int64(h.version) {
		err = PeerIncompatibleVersion
		return
	}

	if their.Genesis != h.genesis {
		err = PeerDifferentGenesis
		return
	}

	level2, err := h.protocol.ReceiveHandshake(their)
	if err != nil {
		return
	}

	fileAddress := extractFileAddress(c.Address(), their.FileAddress)

	if level2 > level {
		level = level2
	}
	peer = NewPeer(their.ID, their.Name, their.Height, their.Head, fileAddress, int(their.Version), c, level, h.protocol)

	return
}

func extractFileAddress(sender net.Addr, fileAddressBytes []byte) (fileAddress string) {
	if len(fileAddressBytes) != 0 {
		var tcp *net.TCPAddr

		if len(fileAddressBytes) == 2 {
			filePort := binary.BigEndian.Uint16(fileAddressBytes)
			var ok bool
			if tcp, ok = sender.(*net.TCPAddr); ok {
				return tcp.IP.String() + ":" + strconv.Itoa(int(filePort))
			}
		} else {
			var ep = new(vnode.EndPoint)

			if err := ep.Deserialize(fileAddressBytes); err == nil {
				if ep.Typ.Is(vnode.HostIP) {
					// verify ip
					var ok bool
					if tcp, ok = sender.(*net.TCPAddr); ok {
						err = netool.CheckRelayIP(tcp.IP, ep.Host)
						if err != nil {
							// invalid ip
							ep.Host = tcp.IP
						}
					}

					fileAddress = ep.String()
				} else {
					tcp, err = net.ResolveTCPAddr("tcp", ep.String())
					if err == nil {
						fileAddress = ep.String()
					}
				}
			}
		}
	}

	return
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
