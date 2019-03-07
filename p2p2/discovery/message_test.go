package discovery

import (
	crand "crypto/rand"
	"fmt"
	"testing"
)

func TestEndPoint_Serialize(t *testing.T) {
	var ep endPoint
	buf, err := ep.serialize()
	if err == nil {
		t.Fatal("should failed, because missing host")
	}

	for hLen := 1; hLen < maxHostLength+1; hLen++ {
		ep.host = make([]byte, hLen)

		_, err = crand.Read(ep.host)
		if err != nil {
			panic(err)
		}

		// host length is not larger than maxHostLength, one byte port
		buf, err = ep.serialize()
		if err != nil {
			t.Fatalf("should notfailed, because host length is %d", hLen)
		}
		// meta(1) + host + port(1)
		if len(buf) != len(ep.host)+2 {
			t.Fatalf("serialize length should be %d, but be %d", len(ep.host)+2, len(buf))
		}
		if int(buf[0]) != len(ep.host)+1<<6 {
			t.Fatalf("meta should be %d, but be %d", 1<<7-1, buf[0])
		}
	}

	// default port
	ep.port = DefaultPort
	buf, err = ep.serialize()
	if err != nil {
		t.Fatalf("should not failed")
	}
	// meta(1) + host
	if len(buf) != len(ep.host)+1 {
		t.Fatalf("port should be omit")
	}
	if int(buf[0]) != len(ep.host) {
		t.Fatalf("meta should only be host length")
	}

	ep.port = 10001
	buf, err = ep.serialize()
	if err != nil {
		t.Fatalf("should not failed")
	}
	if len(buf) != len(ep.host)+3 {
		t.Fatalf("port should be 2 bytes")
	}
	if int(buf[0]) != len(ep.host)+(1<<7) {
		t.Fatalf("meta should be %d, but be %d", len(ep.host)+(1<<7), buf[0])
	}

	// host is too too lang
	ep.host = append(ep.host, maxHostLength+1)
	buf, err = ep.serialize()
	if err == nil {
		t.Fatalf("should failed, because host length is %d larger than %d", len(ep.host), maxHostLength)
	}

	fmt.Println(buf)
}
