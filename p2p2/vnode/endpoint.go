package vnode

import (
	"errors"
	"net"
	"strconv"
)

var errInvalidIP = errors.New("invalid IP")
var errInvalidHost = errors.New("invalid Host")

// EndPoint is the net address format `IP:Port` or `domain:Port`
type EndPoint struct {
	Host []byte
	Port int
	typ  HostType
}

/*
 * EndPoint serialize will not use ProtoBuffers is because we should ensure the neighbors message is short than 1200 bytes
 * but PB is variable-length-encode, the length of encoded []byte is unknown before encode.
 *
 * EndPoint serialize structure
 * +----------+----------------------+-------------+
 * |   Meta   |         Host         |  Port(opt)  |
 * |  1 byte  |      0 ~ 63 bytes    |   2 bytes   |
 * +----------+----------------------+-------------+
 * Meta structure
 * +---------------------+--------+--------+
 * |     Host Length     |  Host  |  Port  |
 * |       6 bits        |  1 bit |  1 bit |
 * +---------------------+--------+--------+
 * Host Length is the byte-count of Host
 * Host: 0 IP. 1 Domain
 * Port: 0 no IP, mean DefaultPort. 1 has 2 bytes Port
 */

func (e EndPoint) Serialize() (buf []byte, err error) {
	hLen := len(e.Host)
	if hLen == 0 {
		err = errMissHost
		return
	}
	if hLen > MaxHostLength {
		err = errInvalidHost
		return
	}

	buf = make([]byte, e.Length())

	// set Host length
	buf[0] |= byte(hLen) << 2

	if e.typ == HostDomain || len(e.Host) > net.IPv6len {
		buf[0] |= 2
	}

	if e.Port != DefaultPort {
		buf[0] |= 1
		buf[len(buf)-1] = byte(e.Port)
		buf[len(buf)-2] = byte(e.Port >> 8)
	}

	copy(buf[1:hLen+1], e.Host)

	return
}

func (e *EndPoint) Deserialize(buf []byte) (err error) {
	if len(buf) == 0 {
		err = errInvalidHost
		return
	}

	hLen := buf[0] >> 4

	if hLen == 0 {
		err = errMissHost
		return
	}

	if buf[0]&2 > 0 {
		e.typ = HostDomain
	} else {
		e.typ = HostIP
	}

	hasPort := buf[0]&1 > 0

	decodeLen := int(hLen) + 1
	if hasPort {
		decodeLen += PortLength
	}

	if len(buf) < decodeLen {
		err = errUnmatchedLength
		return
	}

	e.Host = buf[1 : 1+hLen]

	if hasPort {
		e.Port = int(buf[len(buf)-1]) | (int(buf[len(buf)-2]) << 8)
	} else {
		e.Port = DefaultPort
	}

	return
}

func (e EndPoint) Length() (n int) {
	// meta
	n++

	// Host
	n += len(e.Host)

	// Port
	if e.Port != DefaultPort {
		n += PortLength
	}

	return n
}

func (e EndPoint) String() string {
	return e.Hostname() + ":" + strconv.FormatInt(int64(e.Port), 10)
}

func (e EndPoint) Hostname() string {
	if e.typ == HostDomain {
		return string(e.Host)
	}

	return net.IP(e.Host).String()
}

func parseHost(str string) (buf []byte, hostType HostType, err error) {
	if str == "" {
		err = errMissHost
		return
	}

	if ip := net.ParseIP(str); len(ip) == 0 {
		return []byte(str), HostDomain, nil
	} else if ip4 := ip.To4(); len(ip4) != 0 {
		return ip4, HostIPv4, nil
	} else {
		return ip, HostIPv6, nil
	}
}
