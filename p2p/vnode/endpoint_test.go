package vnode

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/vnode/protos"
)

func TestEndPoint_Serialize(t *testing.T) {
	var ep EndPoint
	buf, err := ep.Serialize()
	if err == nil {
		t.Fatal("should failed, because missing Host")
	}

	for hLen := 1; hLen < MaxHostLength+1; hLen++ {
		ep.Host = make([]byte, hLen)
		// port is 0, so occupy 2 bytes

		_, err = crand.Read(ep.Host)
		if err != nil {
			panic(err)
		}

		// Host length is not larger than maxHostLength
		buf, err = ep.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize endpoint: %v", err)
		}

		// meta(1) + Host + Port(2)
		if len(buf) != len(ep.Host)+3 {
			t.Fatalf("serialize length should be %d, but be %d", len(ep.Host)+2, len(buf))
		}

		meta := len(ep.Host)<<2 + 1
		if len(ep.Host) > net.IPv6len || ep.Typ == HostDomain {
			meta += 2
		}
		if int(buf[0]) != meta {
			t.Fatalf("meta should be %d, but be %d", meta, buf[0])
		}
	}

	// default Port
	ep.Port = DefaultPort
	buf, err = ep.Serialize()
	if err != nil {
		t.Fatalf("should not failed")
	}
	// meta(1) + Host
	if len(buf) != len(ep.Host)+1 {
		t.Fatalf("Port should be omit")
	}

	meta := len(ep.Host) << 2
	if len(ep.Host) > net.IPv6len || ep.Typ == HostDomain {
		meta += 2
	}
	if int(buf[0]) != meta {
		t.Fatalf("meta should only be Host length")
	}

	// Host is too too lang
	ep.Host = append(ep.Host, MaxHostLength+1)
	buf, err = ep.Serialize()
	if err == nil {
		t.Fatalf("should failed, because Host length is %d larger than %d", len(ep.Host), MaxHostLength)
	}
}

func TestEndPoint_Deserialize(t *testing.T) {
	endpoints := []*EndPoint{{
		Host: []byte{0, 0, 0, 0},
		Port: 8888,
		Typ:  HostIPv4,
	}, {
		Host: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Port: 8889,
		Typ:  HostIPv6,
	}, {
		Host: []byte("vite.org"),
		Port: 9000,
		Typ:  HostDomain,
	}, {
		Host: []byte("vite.org"),
		Port: 8483,
		Typ:  HostDomain,
	}}

	var data [][]byte
	for _, ep := range endpoints {
		if buf, err := ep.Serialize(); err != nil {
			t.Error(err)
		} else {
			data = append(data, buf)
		}
	}

	for i, buf := range data {
		var e = new(EndPoint)
		if err := e.Deserialize(buf); err != nil {
			t.Error(err)
		}
		if !endpoints[i].Equal(e) {
			t.Error("not equal")
		}
	}
}

func ExampleParseIP() {
	// even if ip is v4, the parsed IP is 16 bytes
	ip := net.ParseIP("127.0.0.1")

	fmt.Println(len(ip))

	// Output:
	// 16
}

func TestParseEndPoint(t *testing.T) {
	factor := func(host []byte, typ HostType, port int, str string) func(e EndPoint) error {
		return func(e EndPoint) error {
			if !bytes.Equal(e.Host, host) {
				return fmt.Errorf("different host: %v %v", e.Host, host)
			}
			if e.Port != port {
				return fmt.Errorf("different port: %d %d", e.Port, port)
			}
			if e.Typ != typ {
				return fmt.Errorf("different type: %d %d", e.Typ, typ)
			}
			if e.String() != str {
				return fmt.Errorf("should %s get %s", str, e.String())
			}
			return nil
		}
	}

	var protocolTests = [...]struct {
		url    string
		handle func(e EndPoint) error
	}{
		{
			"vite.org",
			factor([]byte("vite.org"), HostDomain, DefaultPort, "vite.org:8483"),
		},
		{
			"vite.org:8888",
			factor([]byte("vite.org"), HostDomain, 8888, "vite.org:8888"),
		},
		{
			"127.0.0.1",
			factor([]byte{127, 0, 0, 1}, HostIPv4, DefaultPort, "127.0.0.1:8483"),
		},
		{
			"127.0.0.1:8888",
			factor([]byte{127, 0, 0, 1}, HostIPv4, 8888, "127.0.0.1:8888"),
		},
		{
			"[127.0.0.1]",
			factor([]byte{127, 0, 0, 1}, HostIPv4, DefaultPort, "127.0.0.1:8483"),
		},
		{
			"[127.0.0.1]:8888",
			factor([]byte{127, 0, 0, 1}, HostIPv4, 8888, "127.0.0.1:8888"),
		},
		{
			"[2001:db8::ff00:42:8329]",
			factor([]byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 255, 0, 0, 66, 131, 41}, HostIPv6, DefaultPort, "[2001:db8::ff00:42:8329]:8483"),
		},
		{
			"[2001:db8::ff00:42:8329]:8888",
			factor([]byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 255, 0, 0, 66, 131, 41}, HostIPv6, 8888, "[2001:db8::ff00:42:8329]:8888"),
		},
	}

	var e EndPoint
	var err error
	for _, tt := range protocolTests {
		e, err = ParseEndPoint(tt.url)
		if err != nil {
			t.Error(err)
		} else {
			if err = tt.handle(e); err != nil {
				t.Error(tt.url, err)
			}
		}
	}
}

func BenchmarkEndPoint_Serialize_PB(b *testing.B) {
	var e EndPoint

	var total int
	for i := 0; i < b.N; i++ {
		n := rand.Intn(MaxHostLength)
		if n == 0 {
			n++
		}
		e.Host = make([]byte, n)

		crand.Read(e.Host)
		e.Port = 8483
		e.Typ = HostIP

		pb := &protos.EndPoint{
			Host:     e.Host,
			Port:     int32(e.Port),
			HostType: int32(e.Typ),
		}

		buf, err := proto.Marshal(pb)
		if err != nil {
			b.Errorf("Failed to marshal protobuf: %v", err)
		}

		total += len(buf)
	}

	fmt.Println("average length", total/b.N)
}

func BenchmarkEndPoint_Serialize(b *testing.B) {
	var e EndPoint

	var total int
	for i := 0; i < b.N; i++ {
		n := rand.Intn(MaxHostLength)
		if n == 0 {
			n++
		}
		e.Host = make([]byte, n)

		crand.Read(e.Host)
		e.Port = 8483
		e.Typ = HostIP

		buf, err := e.Serialize()
		if err != nil {
			b.Errorf("Failed to serialize: %v", err)
		}

		total += len(buf)
	}

	fmt.Println("average length", total/b.N)
}
