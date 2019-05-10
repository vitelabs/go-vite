package p2p

import (
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/p2p/netool"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/p2p/protos"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/log15"
)

const (
	CodeDisconnect  Code = 1
	CodeEncRequest  Code = 2
	CodeEncResponse Code = 3
	CodeHandshake   Code = 4
	CodeControlFlow Code = 5
	CodeHeartBeat   Code = 6

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

const nonceLen = 32
const signatureLen = 64

type HandshakeMsg struct {
	Version uint32

	NetID uint32

	Name string

	ID vnode.NodeID

	Timestamp int64

	Height  uint64
	Head    types.Hash
	Genesis types.Hash

	Key []byte // is producer

	FileAddress []byte
}

func (b *HandshakeMsg) Serialize() (data []byte, err error) {
	pb := &protos.Handshake{
		Version:     b.Version,
		NetId:       b.NetID,
		Name:        b.Name,
		ID:          b.ID.Bytes(),
		Timestamp:   b.Timestamp,
		Height:      b.Height,
		Genesis:     b.Genesis.Bytes(),
		Key:         b.Key,
		FileAddress: b.FileAddress,
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
	b.Height = pb.Height
	b.Genesis, err = types.BytesToHash(pb.Genesis)
	if err != nil {
		return
	}
	b.Key = pb.Key

	b.FileAddress = pb.FileAddress

	return nil
}

type handshaker struct {
	version     uint32
	netId       uint32
	name        string
	id          vnode.NodeID
	genesis     types.Hash
	fileAddress []byte

	priv ed25519.PrivateKey

	protocol Protocol

	log log15.Logger
}

func (h *handshaker) catch(codec Codec, err error) {
	_ = Disconnect(codec, err)
	_ = codec.Close()
}

func (h *handshaker) sendHandshake(codec Codec) (err error) {
	hsm := HandshakeMsg{
		Version:     h.version,
		NetID:       h.netId,
		Name:        h.name,
		ID:          h.id,
		Timestamp:   time.Now().Unix(),
		FileAddress: h.fileAddress,
	}
	hsm.Key, hsm.Height, hsm.Genesis = h.protocol.ProtoData()
	h.genesis = hsm.Genesis

	hspkt, err := hsm.Serialize()
	if err != nil {
		return
	}

	codec.SetWriteTimeout(handshakeTimeout)
	err = codec.WriteMsg(Msg{
		Code:    CodeHandshake,
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

	if msg.Code == CodeDisconnect {
		if len(msg.Payload) > 0 {
			err = PeerError(msg.Payload[0])
		}

		return
	}

	if msg.Code != CodeHandshake {
		return nil, PeerNotHandshakeMsg
	}

	their = new(HandshakeMsg)
	err = their.Deserialize(msg.Payload)
	if err != nil {
		return nil, PeerUnmarshalError
	}

	return
}

func (h *handshaker) doHandshake(codec Codec, level Level) (their *HandshakeMsg, level2 Level, err error) {
	err = h.sendHandshake(codec)
	if err != nil {
		return
	}

	their, err = h.readHandshake(codec)
	if err != nil {
		return
	}

	codec.SetReadTimeout(readMsgTimeout)
	codec.SetWriteTimeout(writeMsgTimeout)

	if their.NetID != h.netId {
		err = PeerDifferentNetwork
		return
	}

	if their.Version < h.version {
		err = PeerIncompatibleVersion
		return
	}

	if their.Genesis != h.genesis {
		err = PeerDifferentGenesis
	}

	level2, err = h.protocol.ReceiveHandshake(their)

	return
}

func (h *handshaker) Handshake(codec Codec, level Level) (peer PeerMux, err error) {
	their, level, err := h.doHandshake(codec, level)

	if err != nil {
		h.catch(codec, err)
		return
	}

	fileAddress := extractFileAddress(codec.Address(), their.FileAddress)

	peer = NewPeer(their.ID, their.Name, their.Height, fileAddress, their.Version, codec, level, h.protocol)

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
