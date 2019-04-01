package vnode

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

var errInvalidIP = errors.New("invalid IP")
var errInvalidHost = errors.New("invalid Host")

// EndPoint is the net address format `IP:Port` or `domain:Port`
type EndPoint struct {
	Host []byte
	Port int
	Typ  HostType
}

// Serialize not use ProtoBuffers, because we should ensure the neighbors message is short than 1200 bytes
// but PB is variable-length-encode, the length of encoded []byte is unknown before encode.
//
// EndPoint serialize structure
// +----------+----------------------+-------------+
// |   Meta   |         Host         |  Port(opt)  |
// |  1 byte  |      0 ~ 63 bytes    |   2 bytes   |
// +----------+----------------------+-------------+
// Meta structure
// +---------------------+--------+--------+
// |     Host Length     |  Host  |  Port  |
// |       6 bits        |  1 bit |  1 bit |
// +---------------------+--------+--------+
// Host Length is the byte-count of Host
// Host: 0 IP. 1 Domain
// Port: 0 no IP, mean DefaultPort. 1 has 2 bytes Port
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

	if e.Typ == HostDomain || len(e.Host) > net.IPv6len {
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

// Deserialize parse []byte to EndPoint, the memory EndPoint should be allocate before.
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
		e.Typ = HostDomain
	} else {
		e.Typ = HostIP
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

// Length return the serialized []byte length
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

// String return `domain:port` or `[IP]:port`
func (e EndPoint) String() string {
	return e.Hostname() + ":" + strconv.FormatInt(int64(e.Port), 10)
}

// Hostname return domain or [IP.String()]
func (e EndPoint) Hostname() string {
	if e.Typ == HostDomain {
		return string(e.Host)
	}

	return "[" + net.IP(e.Host).String() + "]"
}

func parseHost(hostname string) (buf []byte, hostType HostType, err error) {
	if len(hostname) < 2 {
		err = errMissHost
		return
	}

	prefix := hostname[0] == '['
	suffix := hostname[len(hostname)-1] == ']'
	if prefix && suffix {
		hostname = hostname[1 : len(hostname)-1]

		if ip := net.ParseIP(hostname); len(ip) == 0 {
			err = errInvalidHost
			return
		} else if ip4 := ip.To4(); len(ip4) != 0 {
			return ip4, HostIPv4, nil
		} else {
			return ip, HostIPv6, nil
		}
	} else if prefix || suffix {
		err = errInvalidHost
		return
	} else {
		return []byte(hostname), HostDomain, nil
	}
}

// ParseEndPoint parse a string to EndPoint
// host MUST format one of the following styles:
// 1. [IP]:port
// 2. [IP]
// 3. hostname:port
// 4. hostname
func ParseEndPoint(host string) (e EndPoint, err error) {
	if host == "" {
		err = errMissHost
		return
	}

	index := strings.LastIndex(host, "]:")

	var hostname, port string
	if index > 0 {
		// style 1
		hostname = host[:index+1]
		port = host[index+2:]
	} else if strings.HasSuffix(host, "]") {
		// style 2
		hostname = host
		port = ""
	} else if index = strings.LastIndex(host, ":"); index > 0 {
		// style 3
		hostname = host[:index]
		port = host[index+1:]
	} else {
		// style 4
		hostname = host
	}

	e.Host, e.Typ, err = parseHost(hostname)
	if err != nil {
		return
	}

	if port == "" {
		e.Port = DefaultPort
	} else {
		e.Port, err = parsePort(port)
		if err != nil {
			return
		}
	}

	return
}

func FromUDPAddr(addr *net.UDPAddr) (e EndPoint) {
	var ip = addr.IP
	if ip4 := ip.To4(); ip4 != nil {
		e.Host = ip4
		e.Typ = HostIPv4
	} else {
		e.Host = ip
		e.Typ = HostIPv6
	}

	e.Port = addr.Port

	return
}
