package p2p

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/snappy"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func Buffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func EncodeToReader(s Serializable) (r io.Reader, size uint64, err error) {
	data, err := s.Serialize()
	if err != nil {
		return
	}

	return BytesToReader(data), uint64(len(data)), nil
}

func BytesToReader(data []byte) io.Reader {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	n, err := buf.Write(data)
	if err != nil || n != len(data) {
		bufferPool.Put(buf)
		return bytes.NewReader(data)
	}

	return buf
}

func PackMsg(cmdset, cmd uint64, s Serializable) (*Msg, error) {
	r, size, err := EncodeToReader(s)
	if err != nil {
		return nil, err
	}

	return &Msg{
		CmdSet:  cmdset,
		Cmd:     cmd,
		Size:    size,
		Payload: r,
	}, nil
}

type protoMsgRW struct {
	fd           io.ReadWriter
	compressible bool
}

func (prw *protoMsgRW) ReadMsg() (msg Msg, err error) {
	head := make([]byte, headerLength)
	if _, err = io.ReadFull(prw.fd, head); err != nil {
		return
	}

	msg.CmdSet = binary.BigEndian.Uint64(head[:8])
	msg.Cmd = binary.BigEndian.Uint64(head[8:16])
	msg.Size = binary.BigEndian.Uint64(head[16:24])

	if msg.Size > maxPayloadSize {
		err = errMsgTooLarge
		return
	}

	payload := make([]byte, msg.Size)
	n, err := io.ReadFull(prw.fd, payload)
	if err != nil {
		return
	}
	if uint64(n) != msg.Size {
		err = fmt.Errorf("read incomplete message %d/%d\n", n, msg.Size)
		return
	}

	if prw.compressible {
		var fullSize int
		fullSize, err = snappy.DecodedLen(payload)
		if err != nil {
			return
		}

		payload, err = snappy.Decode(nil, payload)
		if err != nil {
			return
		}
		msg.Size = uint64(fullSize)
	}

	msg.Payload = BytesToReader(payload)

	return
}

func (prw *protoMsgRW) WriteMsg(msg Msg) (err error) {
	if msg.Size > maxPayloadSize {
		return errMsgTooLarge
	}

	payload, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return
	}

	// put buffer back in pool
	if r, ok := msg.Payload.(*bytes.Buffer); ok {
		r.Reset()
		bufferPool.Put(r)
	}

	if prw.compressible {
		payload = snappy.Encode(nil, payload)
		msg.Size = uint64(len(payload))
	}

	head := make([]byte, headerLength)
	binary.BigEndian.PutUint64(head[:8], msg.CmdSet)
	binary.BigEndian.PutUint64(head[8:16], msg.Cmd)
	binary.BigEndian.PutUint64(head[16:24], msg.Size)

	// write header
	if _, err = prw.fd.Write(head); err != nil {
		return
	}

	// write payload
	_, err = prw.fd.Write(payload)

	return
}

// protox is the base transport
var handshakeTimeout = 10 * time.Second
var msgReadTimeout = 1 * time.Minute
var msgWriteTimeout = 30 * time.Second

type protox struct {
	fd    net.Conn
	rw    *protoMsgRW
	rLock sync.Mutex
	wLock sync.Mutex
}

func newProtoX(fd net.Conn) *protox {
	fd.SetReadDeadline(time.Now().Add(handshakeTimeout))
	return &protox{
		fd: fd,
	}
}

func (p *protox) ReadMsg() (msg Msg, err error) {
	p.rLock.Lock()
	defer p.rLock.Lock()

	p.fd.SetReadDeadline(time.Now().Add(msgReadTimeout))

	return p.rw.ReadMsg()
}
func (p *protox) WriteMsg(msg Msg) (err error) {
	p.wLock.Lock()
	defer p.wLock.Unlock()

	p.fd.SetWriteDeadline(time.Now().Add(msgWriteTimeout))

	return p.rw.WriteMsg(msg)
}

func (p *protox) Handshake(ours *Handshake) (their *Handshake, err error) {
	errch := make(chan error, 1)

	go func() {
		errch <- Send(p.rw, baseProtocolCmdSet, handshakeCmd, ours)
	}()

	if their, err = readHandshake(p.rw); err != nil {
		<-errch
		return
	}

	if err = <-errch; err != nil {
		return
	}

	if their.Version >= compressibleVersion {
		p.rw.compressible = true
	}

	return their, nil
}

func readHandshake(r MsgReader) (h *Handshake, err error) {
	msg, err := r.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.CmdSet != baseProtocolCmdSet {
		return nil, fmt.Errorf("should be baseProtocolCmdSet, got %x\n", msg.CmdSet)
	}

	payload, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("read handshake payload error: %v\n", err)
	}

	if msg.Cmd == discCmd {
		discReason, err := DeserializeDiscReason(payload)
		if err != nil {
			return nil, fmt.Errorf("disconnected, but parse DiscReason error: %v\n", err)
		}

		return nil, discReason
	}

	if msg.Cmd != handshakeCmd {
		return nil, fmt.Errorf("should be handshake message, but got %x\n", err)
	}

	h = new(Handshake)
	err = h.Deserialize(payload)

	return
}

func (p *protox) close(err error) {
	p.rLock.Lock()
	defer p.rLock.Unlock()

	if p.rw != nil {
		if reason, ok := err.(DiscReason); ok && reason != DiscNetworkError {
			Send(p.rw, baseProtocolCmdSet, discCmd, reason)
		}
	}

	p.fd.Close()
}

func Send(w MsgWriter, cmdset, cmd uint64, s Serializable) error {
	msg, err := PackMsg(cmdset, cmd, s)
	if err != nil {
		return err
	}

	return w.WriteMsg(*msg)
}

func SendMsg(w MsgWriter, msg Msg) error {
	return w.WriteMsg(msg)
}
