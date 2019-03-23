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

package p2p

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/golang/snappy"
)

const headLength = 3
const maxIdLength = 4
const maxPayloadLength = 3
const maxPayloadSize = (1 << (maxPayloadLength * 8)) - 1 // 15MB
const readMsgTimeout = 30 * time.Second
const writeMsgTimeout = 30 * time.Second

var errMsgPayloadTooLarge = errors.New("message payload is too large")
var errWriteTooShort = errors.New("write message too short")

// Codec is an transport can encode messages to bytes, transmit bytes, then decode bytes to messages
type Codec interface {
	MsgReadWriter
	Close() error
	SetReadTimeout(timeout time.Duration)
	SetWriteTimeout(timeout time.Duration)
	SetTimeout(timeout time.Duration)
	Address() string
}

type CodecFactory interface {
	CreateCodec(conn net.Conn) Codec
}

/*
 * message structure
 * |------------ head --------------|
 * +----------+----------+----------+------------------+----------------------+----------------------------+
 * |   Meta   |  ProtoID |   Code   |     Id (opt)     | Payload Length (opt) |       Payload (opt)        |
 * |  1 byte  |  1 byte  |  1 byte  |    0 ~ 4 bytes   |     0 ~ 3 bytes      |         0 ~ 15 MB          |
 * +----------+----------+----------+------------------+----------------------+----------------------------+
 *
 * Meta structure
 * +-------------+-------------+----------+-----------------+
 * |   Id size   | Length size | Compress |    Reserved     |
 * |   2 bits    |    2 bits   |  1 bit   |     3 bits      |
 * +-------------+-------------+----------+-----------------+
 * Length size: the bytes-number of `Payload Length` field, min 0 bytes ~ max 3 bytes
 * Id size: the bytes-number of `Id size` field: 0 bytes, 1 byte, 2 bytes, 4 bytes
 * Compress: 0 no compressed, 1 compressed
 */

func putIdSize(idLength byte) byte {
	switch idLength {
	case 4:
		return 3
	default:
		return idLength
	}
}

func getIdSize(code byte) byte {
	switch code {
	case 3:
		return 4
	default:
		return code
	}
}

func retrieveMeta(meta byte) (isize, lsize byte, compressed bool) {
	isize = getIdSize(meta >> 6)
	lsize = meta << 2 >> 6
	compressed = (meta << 4 >> 7) > 0
	return
}

func storeMeta(isize, lsize byte, compressed bool) (meta byte) {
	meta |= putIdSize(isize << 6)
	meta |= lsize << 4
	if compressed {
		meta |= 8
	}
	return
}

type transport struct {
	net.Conn
	readTimeout       time.Duration
	writeTimeout      time.Duration
	minCompressLength int // will not compress message payload if small than minCompressLength bytes
}

func (t *transport) Address() string {
	return t.Conn.RemoteAddr().String()
}

type transportFactory struct {
	minCompressLength int
	readTimeout       time.Duration
	writeTimeout      time.Duration
}

func (tf *transportFactory) CreateCodec(conn net.Conn) Codec {
	return newTransport(conn, tf.minCompressLength, tf.readTimeout, tf.writeTimeout)
}

func newTransport(conn net.Conn, minCompressLength int, readTimeout, writeTimeout time.Duration) Codec {
	return &transport{
		Conn:              conn,
		minCompressLength: minCompressLength,
		readTimeout:       readTimeout,
		writeTimeout:      writeTimeout,
	}
}

func (t *transport) SetReadTimeout(timeout time.Duration) {
	t.readTimeout = timeout
}

func (t *transport) SetWriteTimeout(timeout time.Duration) {
	t.writeTimeout = timeout
}

func (t *transport) SetTimeout(timeout time.Duration) {
	t.readTimeout = timeout
	t.writeTimeout = timeout
}

func (t *transport) ReadMsg() (msg Msg, err error) {
	_ = t.SetReadDeadline(time.Now().Add(t.readTimeout))

	buf := make([]byte, 4)

	_, err = io.ReadFull(t.Conn, buf[:headLength])
	if err != nil {
		return
	}

	meta := buf[0]
	msg.Pid = buf[1]
	msg.Code = buf[2]
	msg.ReceivedAt = time.Now()

	isize, lsize, compressed := retrieveMeta(meta)

	// retrieve id
	if isize > 0 {
		_, err = io.ReadFull(t.Conn, buf[:isize])
		if err != nil {
			return
		}
		msg.Id = uint32(Varint(buf[:isize]))
	}

	// retrieve payload
	if lsize > 0 {
		_, err = io.ReadFull(t.Conn, buf[:lsize])
		if err != nil {
			return
		}

		length := Varint(buf[:lsize])
		if length > maxPayloadSize {
			return msg, errMsgPayloadTooLarge
		}

		msg.Payload = make([]byte, length)
		_, err = io.ReadFull(t.Conn, msg.Payload)
		if err != nil {
			return
		}
	}

	if compressed {
		msg.Payload, err = snappy.Decode(nil, msg.Payload)
		if err != nil {
			return
		}
	}

	return
}

func (t *transport) WriteMsg(msg Msg) (err error) {
	_ = t.SetWriteDeadline(time.Now().Add(t.writeTimeout))

	head := make([]byte, headLength+maxIdLength+maxPayloadLength)
	head[1] = msg.Pid
	head[2] = msg.Code

	var headLen = 3

	var n int
	if msg.Id > 0 {
		n = PutVarint(head[3:], uint(msg.Id), maxIdLength)
		head[0] |= putIdSize(byte(n)) << 6
		headLen += n
	}

	payloadLen := len(msg.Payload)
	if payloadLen > t.minCompressLength {
		payload := snappy.Encode(nil, msg.Payload)
		if len(payload) < payloadLen {
			msg.Payload = payload
			head[0] |= 8 // compressed
			payloadLen = len(payload)
		}
	}

	if payloadLen > 0 {
		n = PutVarint(head[headLen:], uint(payloadLen), maxPayloadLength)
		head[0] |= byte(n << 4)
		headLen += n
	}

	data := make([]byte, headLen+payloadLen)
	copy(data, head[:headLen])
	copy(data[headLen:], msg.Payload)

	n, err = t.Conn.Write(data)
	if err != nil {
		return
	}

	if n != len(data) {
		return errWriteTooShort
	}

	return
}

func Varint(buf []byte) (n uint) {
	t := len(buf)
	for i := 0; i < t; i++ {
		n |= uint(buf[i]) << (uint(t-i-1) * 8)
	}

	return
}

func PutVarint(buf []byte, n uint, maxBytes int) (m int) {
	_ = buf[maxBytes-1]

	for m = 1; m < maxBytes+1; m++ {
		if n>>uint(m*8) == 0 {
			break
		}
	}

	for i := 0; i < m; i++ {
		buf[i] = byte(n >> (uint(m-i-1) * 8))
	}

	return
}
