package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/network"
)

type P2PVersion = uint32

const Version P2PVersion = 1

const baseProtocolCmdSet = 0
const handshakeCmd = 0
const discCmd = 1

const headerLength = 32
const maxPayloadSize = ^uint32(0) >> 8 // 15MB
const shakeTimeout = 10 * time.Second

// head message is the first message in chain tcp connection
type headMsg struct {
	Version P2PVersion
	NetID   network.ID
}

const headMsgLen = 32 // netId[4] + version[4]

func readHead(conn net.Conn) (head *headMsg, err error) {
	conn.SetReadDeadline(time.Now().Add(shakeTimeout))
	defer conn.SetReadDeadline(time.Time{})

	headPacket := make([]byte, headMsgLen)
	_, err = io.ReadFull(conn, headPacket)
	if err != nil {
		return
	}

	head = new(headMsg)
	head.NetID = network.ID(binary.BigEndian.Uint32(headPacket[:4]))
	head.Version = binary.BigEndian.Uint32(headPacket[4:8])

	return
}

func writeHead(conn net.Conn, head *headMsg) error {
	conn.SetWriteDeadline(time.Now().Add(shakeTimeout))
	defer conn.SetWriteDeadline(time.Time{})

	headPacket := make([]byte, headMsgLen)
	binary.BigEndian.PutUint32(headPacket[:4], uint32(head.NetID))
	binary.BigEndian.PutUint32(headPacket[4:8], head.Version)

	if n, err := conn.Write(headPacket); err != nil {
		return err
	} else if n != headMsgLen {
		return fmt.Errorf("write incomplete HeadMsg %d/%d", n, headMsgLen)
	}

	return nil
}

func headShake(conn net.Conn, head *headMsg) (their *headMsg, err error) {
	send := make(chan error, 1)
	common.Go(func() {
		send <- writeHead(conn, head)
	})

	if their, err = readHead(conn); err != nil {
		return
	}

	if err = <-send; err != nil {
		return
	}

	return
}

//var msgPool = &sync.Pool{
//	New: func() interface{} {
//		return &Msg{}
//	},
//}

func NewMsg() *Msg {
	//return msgPool.Get().(*Msg)
	return &Msg{}
}

func PackMsg(cmdSetId CmdSet, cmd Cmd, id uint64, s Serializable) (*Msg, error) {
	data, err := s.Serialize()
	if err != nil {
		return nil, err
	}

	size := uint32(len(data))
	if size > maxPayloadSize {
		return nil, errMsgTooLarge
	}

	msg := NewMsg()
	msg.CmdSet = cmdSetId
	msg.Cmd = cmd
	msg.Payload = data
	msg.Id = id

	return msg, nil
}

func ReadMsg(reader io.Reader) (msg *Msg, err error) {
	head := make([]byte, headerLength)

	if _, err = io.ReadFull(reader, head); err != nil {
		return
	}

	msg = new(Msg)
	msg.CmdSet = binary.BigEndian.Uint32(head[:4])
	msg.Cmd = binary.BigEndian.Uint16(head[4:6])
	msg.Id = binary.BigEndian.Uint64(head[6:14])
	size := binary.BigEndian.Uint32(head[14:18])

	if size > maxPayloadSize {
		return nil, errMsgTooLarge
	}

	payload := make([]byte, size)
	if _, err = io.ReadFull(reader, payload); err != nil {
		return
	}

	msg.Payload = payload
	msg.SendAt = time.Unix(int64(binary.BigEndian.Uint64(head[18:26])), 0)
	msg.ReceivedAt = time.Now()

	return
}

func WriteMsg(writer io.Writer, msg *Msg) (err error) {
	defer msg.Recycle()

	size := uint32(len(msg.Payload))

	if size == 0 {
		return errMsgNull
	}

	if size > maxPayloadSize {
		return errMsgTooLarge
	}

	head := make([]byte, headerLength)
	binary.BigEndian.PutUint32(head[:4], msg.CmdSet)
	binary.BigEndian.PutUint16(head[4:6], msg.Cmd)
	binary.BigEndian.PutUint64(head[6:14], msg.Id)
	binary.BigEndian.PutUint32(head[14:18], size)
	binary.BigEndian.PutUint64(head[18:26], uint64(time.Now().Unix()))

	// write header
	var n int
	if n, err = writer.Write(head); err != nil {
		return
	} else if n != headerLength {
		return fmt.Errorf("write incomplement message header %d/%d bytes", n, headerLength)
	}

	// write payload
	if n, err = writer.Write(msg.Payload); err != nil {
		return
	} else if uint32(n) != size {
		return fmt.Errorf("write incomplement message payload %d/%d bytes", n, size)
	}

	return
}

var errHandshakeVerify = errors.New("signature of handshake Msg verify failed")
var errHandshakeNotComp = errors.New("handshake payload is too small, maybe old version")

func readHandshake(r io.Reader) (h *Handshake, err error) {
	msg, err := ReadMsg(r)
	if err != nil {
		return nil, err
	}
	if msg.CmdSet != baseProtocolCmdSet {
		return nil, fmt.Errorf("should be baseProtocolCmdSet, got %x", msg.CmdSet)
	}

	if msg.Cmd == discCmd {
		discReason, err := DeserializeDiscReason(msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("disconnected, but parse DiscReason error: %v", err)
		}

		return nil, discReason
	}

	if msg.Cmd != handshakeCmd {
		return nil, fmt.Errorf("should be handshake message, but got %x", err)
	}

	if len(msg.Payload) < 64 {
		return nil, errHandshakeNotComp
	}

	h = new(Handshake)
	err = h.Deserialize(msg.Payload[64:])
	if err != nil {
		return
	}

	if !ed25519.Verify(h.ID[:], msg.Payload[64:], msg.Payload[:64]) {
		return nil, errHandshakeVerify
	}

	return
}
