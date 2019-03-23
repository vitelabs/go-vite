package vnode

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/p2p2/vnode/protos"
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

func ExampleParseIP() {
	// even if ip is v4, the parsed IP is 16 bytes
	ip := net.ParseIP("127.0.0.1")

	fmt.Println(len(ip))

	// Output:
	// 16
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
