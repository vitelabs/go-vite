package net

import (
	"encoding/binary"
	"errors"
	"io"
	net2 "net"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/vite/net/message"
)

var errReadTooShort = errors.New("read too short")
var errWriteTooShort = errors.New("write too short")

type syncCode byte

const (
	syncHandshake syncCode = iota
	syncHandshakeDone
	syncHandshakeErr
	syncRequest
	syncReady // server begin transmit data
	syncMissing
	syncNoAuth
	syncServerError // server error, like open reader failed
	syncQuit
)

type syncMsg struct {
	code    syncCode
	payload interface{}
}

type syncConnection interface {
	net2.Conn
	ID() peerId
	download(from, to uint64) (err error)
	read() (syncMsg, error)
	write(syncMsg) error
	speed() float64
	status() FileConnStatus
	isBusy() bool
	height() uint64
}

type syncConnectionFactory interface {
	syncConnInitiator
	syncConnReceiver
}

type syncConnReceiver interface {
	receive(conn net2.Conn) (syncConnection, error)
}

type syncConnInitiator interface {
	initiate(conn net2.Conn, peer downloadPeer) (syncConnection, error)
}

type defaultSyncConnectionFactory struct {
	// todo handshake and codec
	chain syncCacher
	peers *peerSet
}

func (d *defaultSyncConnectionFactory) initiate(conn net2.Conn, peer downloadPeer) (syncConnection, error) {
	buf := make([]byte, 1+len(peerId{}))

	buf[0] = byte(syncHandshake)
	copy(buf[1:], peer.ID().Bytes())

	_, err := conn.Write(buf)
	if err != nil {
		return nil, err
	}

	_, err = conn.Read(buf[:1])
	if err != nil {
		return nil, err
	}

	if buf[0] != byte(syncHandshakeDone) {
		return nil, errors.New("handshake error")
	}

	return &syncConn{
		Conn:         conn,
		downloadPeer: peer,
		cacher:       d.chain,
		buf:          make([]byte, 1024),
	}, nil
}

func (d *defaultSyncConnectionFactory) receive(conn net2.Conn) (syncConnection, error) {
	var id peerId
	buf := make([]byte, 1+len(id))

	_, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	if buf[0] != byte(syncHandshake) {
		return nil, errors.New("not handshake message")
	}

	_, err = conn.Write([]byte{byte(syncHandshakeDone)})
	if err != nil {
		return nil, err
	}

	copy(id[:], buf[1:])

	p := d.peers.get(id)
	if p == nil {
		return nil, errors.New("no auth")
	}

	return &syncConn{
		Conn:         conn,
		downloadPeer: p,
	}, nil
}

type FileConnStatus struct {
	Id    string
	Addr  string
	Speed float64
}

type syncConn struct {
	net2.Conn
	downloadPeer
	busy   int32   // atomic
	t      int64   // timestamp
	_speed float64 // download speed, byte/s
	closed int32
	cacher syncCacher
	buf    []byte
}

func (f *syncConn) speed() float64 {
	return f._speed
}

func (f *syncConn) read() (msg syncMsg, err error) {
	buf := make([]byte, 20)
	n, err := f.Conn.Read(buf)
	if err != nil {
		return
	}
	if n < 20 {
		err = errReadTooShort
		return
	}

	msg.code = syncCode(buf[0])
	if msg.code == syncRequest {
		msg.payload = message.GetChunk{
			Start: binary.BigEndian.Uint64(buf[1:9]),
			End:   binary.BigEndian.Uint64(buf[9:17]),
		}
	}
	return
}

func (f *syncConn) write(msg syncMsg) error {
	buf := make([]byte, 20)
	buf[0] = byte(msg.code)

	if msg.code == syncRequest {
		chunk := msg.payload.(message.GetChunk)
		binary.BigEndian.PutUint64(buf[1:9], chunk.Start)
		binary.BigEndian.PutUint64(buf[9:17], chunk.End)
	}

	_, err := f.Conn.Write(buf)
	return err
}

func (f *syncConn) status() FileConnStatus {
	return FileConnStatus{
		Id:    f.ID().String(),
		Addr:  f.RemoteAddr().String(),
		Speed: f._speed,
	}
}

func (f *syncConn) isBusy() bool {
	return atomic.LoadInt32(&f.busy) == 1
}

func (f *syncConn) download(from, to uint64) (err error) {
	f.setBusy()
	defer f.idle()

	err = f.write(syncMsg{
		code: syncRequest,
		payload: message.GetChunk{
			Start: from,
			End:   to,
		},
	})

	if err != nil {
		return err
	}

	var msg syncMsg
	msg, err = f.read()
	if err != nil {
		return
	}

	if msg.code != syncReady {
		return errors.New("remote not ready")
	}

	cache := f.cacher.GetSyncCache()
	writer, err := cache.NewWriter(from, to)
	if err != nil {
		return err
	}

	defer writer.Close()

	start := time.Now()
	var nr, nw, total int
	var werr error
	for {
		//_ = f.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		nr, err = f.Conn.Read(f.buf)
		total += nr

		nw, werr = writer.Write(f.buf[:nr])
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else if werr != nil || nw != nr {
			return errWriteTooShort
		}
	}

	f._speed = float64(total) / (time.Now().Sub(start).Seconds() + 1)

	return nil
}

func (f *syncConn) setBusy() {
	atomic.StoreInt32(&f.busy, 1)
	atomic.StoreInt64(&f.t, time.Now().Unix())
}

func (f *syncConn) idle() {
	atomic.StoreInt32(&f.busy, 0)
	atomic.StoreInt64(&f.t, time.Now().Unix())
}

func (f *syncConn) close() error {
	if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		return f.Conn.Close()
	}

	return errFileConnClosed
}
