package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/network"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type P2PVersion = uint32

const Version P2PVersion = 0

const baseProtocolCmdSet = 0
const handshakeCmd = 0
const discCmd = 1

const headerLength = 20
const maxPayloadSize = ^uint32(0) >> 8 // 15MB

// bigEnd
func PutUint24(buf []byte, v uint32) {
	if len(buf) < 3 {
		panic("put uint24: target byte slice is not long enough")
	}

	buf[0] = byte(v >> 16)
	buf[1] = byte(v >> 8)
	buf[2] = byte(v)
}

func Uint24(buf []byte) uint32 {
	if len(buf) < 3 {
		panic("read uint24: target byte slice is not long enough")
	}

	return uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])
}

// head message is the first message in a tcp connection
type headMsg struct {
	Version P2PVersion
	NetID   network.ID
}

const headMsgLen = 32 // netId[4] + version[4]

func readHead(conn net.Conn) (head *headMsg, err error) {
	headPacket := make([]byte, headMsgLen)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
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
	headPacket := make([]byte, headMsgLen)
	binary.BigEndian.PutUint32(headPacket[:4], uint32(head.NetID))
	binary.BigEndian.PutUint32(headPacket[4:8], head.Version)

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
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

// @section AsyncMsgConn
var msgReadTimeout = 40 * time.Second
var msgWriteTimeout = 20 * time.Second

const writeBufferLen = 20

type AsyncMsgConn struct {
	fd      net.Conn
	handler func(msg *Msg)
	term    chan struct{}
	wg      sync.WaitGroup
	wqueue  chan *Msg
	errored int32      // atomic, indicate whehter there is an error, readErr(1), writeErr(2)
	errch   chan error // report errch to upper layer
	speed   int        // transport speed, bytes/ns
}

// create an AsyncMsgConn, fd is the basic connection
func NewAsyncMsgConn(fd net.Conn) *AsyncMsgConn {
	return &AsyncMsgConn{
		fd:     fd,
		term:   make(chan struct{}),
		wqueue: make(chan *Msg, writeBufferLen),
		errch:  make(chan error),
	}
}

func (c *AsyncMsgConn) Start() {
	c.wg.Add(1)
	common.Go(c.readLoop)

	c.wg.Add(1)
	common.Go(c.writeLoop)
}

func (c *AsyncMsgConn) Close() {
	select {
	case <-c.term:
	default:
		close(c.term)
		c.wg.Wait()
	}
}

func (c *AsyncMsgConn) report(t int32, err error) {
	if atomic.CompareAndSwapInt32(&c.errored, 0, t) {
		c.errch <- err
	}
}

func (c *AsyncMsgConn) readLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.term:
			return
		default:
			c.fd.SetReadDeadline(time.Now().Add(msgReadTimeout))
			if msg, err := ReadMsg(c.fd); err == nil {
				monitor.LogEvent("p2p_ts", "read")
				monitor.LogDuration("p2p_ts", "read_bytes", int64(len(msg.Payload)))

				c.handler(msg)
			} else {
				fmt.Println("read error", err)
				c.report(1, err)
				return
			}
		}
	}
}

func (c *AsyncMsgConn) writeLoop() {
	defer c.wg.Done()
	defer c.fd.Close()

loop:
	for {
		select {
		case <-c.term:
			break loop
		case msg := <-c.wqueue:
			c.fd.SetWriteDeadline(time.Now().Add(msgWriteTimeout))
			if err := WriteMsg(c.fd, msg); err != nil {
				c.report(2, err)
				return
			}
			monitor.LogEvent("p2p_ts", "write")
			monitor.LogDuration("p2p_ts", "write_bytes", int64(len(msg.Payload)))
		}
	}

	// no error, disconnected initiative
	if atomic.LoadInt32(&c.errored) == 0 {
		for i := 0; i < len(c.wqueue); i++ {
			if err := WriteMsg(c.fd, <-c.wqueue); err != nil {
				return
			}
		}
	}
}

var errTSerrored = errors.New("transport has an error")
var errTSclosed = errors.New("transport has closed")

// send a message asynchronously, put message into a internal buffered channel before send it
// if the internal channel is full, return false
func (c *AsyncMsgConn) SendMsg(msg *Msg) error {
	if atomic.LoadInt32(&c.errored) != 0 {
		return errTSerrored
	}

	select {
	case <-c.term:
		return errTSclosed
	case c.wqueue <- msg:
		return nil
	}
}

// send Handshake data, after signature with ed25519 algorithm
func (c *AsyncMsgConn) Handshake(key ed25519.PrivateKey, our *Handshake) (their *Handshake, err error) {
	data, err := our.Serialize()
	if err != nil {
		return
	}
	sig := ed25519.Sign(key, data)
	// unshift signature before data
	data = append(sig, data...)

	send := make(chan error, 1)
	common.Go(func() {
		msg := NewMsg()
		msg.CmdSet = baseProtocolCmdSet
		msg.Cmd = handshakeCmd
		msg.Payload = data

		send <- WriteMsg(c.fd, msg)
	})

	if their, err = readHandshake(c.fd); err != nil {
		return
	}

	if err = <-send; err != nil {
		return
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
