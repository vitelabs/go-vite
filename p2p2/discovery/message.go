package discovery

import (
	"errors"
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/p2p2/vnode"
)

var errInvalidEndPointBytes = errors.New("invalid endpoint bytes")
var errMissingHost = errors.New("missing host")
var errHostTooLong = fmt.Errorf("host should be short than %d bytes", maxHostLength)

type code byte

const (
	pingCode    code = iota // ping a node to check it alive, also announce self, expect to get a pingAck or pong
	pingAckCode             // pingAck is response to ping and check peer`s auth, expect to get a pong
	pongCode                // response to ping/pingAck
	findNodeCode
	neighborsCode
)

const messageTimeout = 5 * time.Second

const maxHostLength = 1<<7 - 1

func portLen(port int) int {
	if port>>8 == 0 {
		return 1
	}

	return 2
}

type endPoint struct {
	host []byte
	port int
}

/*
 * endPoint serialize will not use ProtoBuffers is because we should ensure the neighbors message is short than 1200 bytes
 * but PB is variable-length-encode, the length of encoded []byte is unknown before encode.
 *
 * endPoint serialize structure
 * +----------+----------------------+-------------+
 * |   Meta   |         Host         |    Port     |
 * |  1 byte  |     0 ~ 127 bytes    | 1 ~ 2 bytes |
 * +----------+----------------------+-------------+
 * Meta structure
 * +-----------+---------------------+
 * | IP Length |     Host Length     |
 * |   2 bits  |       6 bits        |
 * *-----------+---------------------+
 */

func (e endPoint) serialize() (buf []byte, err error) {
	hLen := len(e.host)
	if hLen == 0 {
		err = errMissingHost
		return
	}
	if hLen > maxHostLength {
		err = errHostTooLong
		return
	}

	buf = make([]byte, e.length())

	buf[0] |= byte(hLen)

	if e.port != DefaultPort {
		if e.port < 256 {
			buf[0] |= 1 << 6
			buf[len(buf)-1] = byte(e.port)
		} else {
			buf[0] |= 1 << 7
			buf[len(buf)-1] = byte(e.port)
			buf[len(buf)-2] = byte(e.port >> 8)
		}
	}

	copy(buf[1:], e.host)

	return
}

func desEndPoint(buf []byte) (e endPoint, err error) {
	if len(buf) == 0 {
		err = errInvalidEndPointBytes
		return
	}

	hLen := buf[0] << 2 >> 4

	if hLen == 0 {
		err = errMissingHost
		return
	}

	pLen := buf[0] >> 6
	if len(buf) < int(hLen)+int(pLen)+1 {
		err = errInvalidEndPointBytes
		return
	}
	e.host = buf[1 : 1+hLen]
	if pLen == 0 {
		e.port = DefaultPort
	} else if pLen == 2 {
		e.port = int(buf[len(buf)-1]) | int(buf[len(buf)-2]<<8)
	} else {
		e.port = int(buf[len(buf)-1])
	}

	return
}

func (e endPoint) length() (n int) {
	// meta
	n++

	// host
	n += len(e.host)

	// port
	if e.port != DefaultPort {
		n += portLen(e.port)
	}

	return n
}

type ping struct {
	from      vnode.Node
	to        endPoint
	timestamp int64
	nonce     []byte
	ext       []byte
}

type pong struct {
	id        vnode.NodeID
	to        endPoint
	timestamp int64
	ext       []byte
	ack       []byte
}

type findNode struct {
	id        vnode.NodeID
	target    vnode.NodeID
	count     int
	timestamp int64
}

type neighbors struct {
	id        vnode.NodeID
	timestamp int
	neighbors []endPoint
}
