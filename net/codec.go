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
	"errors"
	"fmt"
	"io"
	_net "net"
	"time"

	"github.com/golang/snappy"
)

const maxPayloadLength = 3
const maxPayloadSize = (1 << (maxPayloadLength * 8)) - 1 // 15MB
const readMsgTimeout = 30 * time.Second
const writeMsgTimeout = 30 * time.Second

var errMsgPayloadTooLarge = errors.New("message payload is too large")

// Codec is an transport can encode messages to bytes, transmit bytes, then decode bytes to messages
type Codec interface {
	MsgReadWriter
	Close() error
	SetReadTimeout(timeout time.Duration)
	SetWriteTimeout(timeout time.Duration)
	SetTimeout(timeout time.Duration)
	Address() _net.Addr
}

type CodecFactory interface {
	CreateCodec(conn _net.Conn) Codec
}

/*
 * message structure
 *  |------- head --------|
 *  +----------+----------+------------------+----------------------+----------------------------+
 *  |   Meta   |   Code   |     Id (opt)     | Payload Length (opt) |       Payload (opt)        |
 *  |  1 byte  |  1 byte  |   0 1 2 4 bytes  |     0 ~ 3 bytes      |         0 ~ 15 MB          |
 *  +----------+----------+------------------+----------------------+----------------------------+
 *
 * Meta structure
 *  +-------------+-------------+----------+-----------------+
 *  |   Id size   | Length size | Compress |    Reserved     |
 *  |   2 bits    |    2 bits   |  1 bit   |     3 bits      |
 *  +-------------+-------------+----------+-----------------+
 * Length size: the bytes-number of `Payload Length` field, min 0 bytes ~ max 3 bytes
 * Id size: the bytes-number of `Id size` field: 0 bytes, 1 byte, 2 bytes, 4 bytes
 * Compress: 0 no compressed, 1 compressed
 */

// idLength to bits
//  0 --> 00
//  1 --> 01
//  2 --> 10
//  4 --> 11
func idLengthToBits(idLength byte) byte {
	switch idLength {
	case 4:
		return 3
	default:
		return idLength
	}
}

// bits to id length
// 00 --> 0
// 01 --> 1
// 10 --> 2
// 11 --> 4
func bitsToIdLength(bits byte) byte {
	switch bits {
	case 3:
		return 4
	default:
		return bits
	}
}

// buf should not small than 4 bytes
func putId(id MsgId, buf []byte) (n byte) {
	if id == 0 {
		return 0
	}

	if id > 65535 {
		buf[0] = byte(id >> 24)
		buf[1] = byte(id >> 16)
		buf[2] = byte(id >> 8)
		buf[3] = byte(id)

		return 4
	}

	if id > 255 {
		buf[0] = byte(id >> 8)
		buf[1] = byte(id)

		return 2
	}

	buf[0] = byte(id)
	return 1
}

func retrieveMeta(meta byte) (isize, lsize byte, compressed bool) {
	isize = bitsToIdLength(meta >> 6)
	lsize = meta << 2 >> 6
	compressed = (meta << 4 >> 7) > 0
	return
}

func storeMeta(isize, lsize byte, compressed bool) (meta byte) {
	meta |= idLengthToBits(isize) << 6
	meta |= lsize << 4
	if compressed {
		meta |= 8
	}
	return
}

type transport struct {
	_net.Conn
	readTimeout       time.Duration
	writeTimeout      time.Duration
	minCompressLength int // will not compress message payload if small than minCompressLength bytes
	readHeadBuf       [4]byte
	writeHeadBuf      [9]byte
	writeBuf          []byte
}

func (t *transport) Address() _net.Addr {
	return t.Conn.RemoteAddr()
}

type transportFactory struct {
	minCompressLength int
	readTimeout       time.Duration
	writeTimeout      time.Duration
}

func (tf *transportFactory) CreateCodec(conn _net.Conn) Codec {
	return NewTransport(conn, tf.minCompressLength, tf.readTimeout, tf.writeTimeout)
}

func NewTransport(conn _net.Conn, minCompressLength int, readTimeout, writeTimeout time.Duration) Codec {
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

// ReadMsg is NOT thread-safe
func (t *transport) ReadMsg() (msg Msg, err error) {
	_ = t.SetReadDeadline(time.Now().Add(t.readTimeout))

	buf := t.readHeadBuf[:]
	_, err = io.ReadFull(t.Conn, buf[:2])
	if err != nil {
		err = fmt.Errorf("failed to read message meta: %v", err)
		return
	}

	meta := buf[0]
	msg.Code = buf[1]

	isize, lsize, compressed := retrieveMeta(meta)

	// retrieve id
	if isize > 0 {
		_, err = io.ReadFull(t.Conn, buf[:isize])
		if err != nil {
			err = fmt.Errorf("failed to read message id: %v", err)
			return
		}
		msg.Id = MsgId(Varint(buf[:isize]))
	}

	// retrieve payload
	if lsize > 0 {
		_, err = io.ReadFull(t.Conn, buf[:lsize])
		if err != nil {
			err = fmt.Errorf("failed to read message length: %v", err)
			return
		}

		length := Varint(buf[:lsize])
		if length > maxPayloadSize {
			err = errMsgPayloadTooLarge
			return
		}

		msg.Payload = make([]byte, length)
		_, err = io.ReadFull(t.Conn, msg.Payload)
		if err != nil {
			err = fmt.Errorf("failed to read message payload: %v", err)
			return
		}
	}

	if compressed {
		var payloadReadLength int
		payloadReadLength, err = snappy.DecodedLen(msg.Payload)
		if err != nil {
			err = fmt.Errorf("failed to decode message length: %v", err)
			return
		}

		payloadUncompressed := make([]byte, payloadReadLength)
		payloadUncompressed, err = snappy.Decode(payloadUncompressed, msg.Payload)

		if err != nil {
			err = fmt.Errorf("failed to decode message payload: %v", err)
			return
		}
		msg.Payload = payloadUncompressed
	}

	return
}

// WriteMsg is NOT thread-safe
func (t *transport) WriteMsg(msg Msg) (err error) {
	_ = t.SetWriteDeadline(time.Now().Add(t.writeTimeout))

	head := t.writeHeadBuf[:]
	head[1] = msg.Code

	var headLen byte = 2

	// store msg id
	isize := putId(msg.Id, head[2:])
	headLen += isize

	// compress payload
	var compress bool
	payloadLen := len(msg.Payload)
	if payloadLen > t.minCompressLength {
		preCompressLength := snappy.MaxEncodedLen(payloadLen)
		if cap(t.writeBuf) < preCompressLength {
			t.writeBuf = make([]byte, preCompressLength)
		}

		payloadCompressed := t.writeBuf[:preCompressLength]
		payloadCompressed = snappy.Encode(payloadCompressed, msg.Payload)

		// smaller after compressed
		if len(payloadCompressed) < payloadLen {
			msg.Payload = payloadCompressed
			payloadLen = len(payloadCompressed)
			compress = true
		}
	}

	// store msg length
	lsize := PutVarint(head[headLen:], uint(payloadLen))
	headLen += lsize

	head[0] = storeMeta(isize, lsize, compress)

	var wsize int
	// send head
	wsize, err = t.Conn.Write(head[:headLen])
	if err != nil {
		return
	}
	if wsize != int(headLen) {
		return errWriteTooShort
	}

	// send payload
	wsize, err = t.Conn.Write(msg.Payload)
	if err != nil {
		return
	}
	if wsize != payloadLen {
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

func PutVarint(buf []byte, n uint) (m byte) {
	if n == 0 {
		return
	}

	for m = 1; (n >> uint(m*8)) > 0; m++ {
	}

	for i := byte(0); i < m; i++ {
		buf[i] = byte(n >> (uint(m-i-1) * 8))
	}

	return
}
