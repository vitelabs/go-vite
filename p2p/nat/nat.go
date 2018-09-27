package nat

import (
	"github.com/syaka-yin/go-nat"
	"github.com/vitelabs/go-vite/log15"
	"net"
	"sync"
	"time"
)

var natLog = log15.New("module", "p2p/nat")

var client nat.NAT
var lock sync.Mutex
var defaultLifetime = 15 * time.Minute

type Addr struct {
	Proto string
	IP    net.IP
	Port  int
}

func isValidIP(ip net.IP) bool {
	return !(ip.IsMulticast() || ip.IsLoopback() || ip.IsUnspecified())
}

func (a *Addr) IsValid() bool {
	return a.IP != nil && isValidIP(a.IP) && a.Port != 0
}

func New() (clt nat.NAT, err error) {
	lock.Lock()
	defer lock.Unlock()

	if client == nil {
		if clt, err = nat.DiscoverGateway(); err == nil {
			client = clt
		}
	}

	return client, err
}

func Map(stop <-chan struct{}, protocol string, lPort, ePort int, desc string, lifetime time.Duration, callback func(*Addr)) {
	if lifetime == 0 {
		lifetime = defaultLifetime
	}

	timer := time.NewTimer(lifetime)
	defer timer.Stop()

	try := 0
	addr := new(Addr)

	for {
		if try >= 3 {
			return
		}

		_, err := New()

		if err != nil {
			time.Sleep(time.Second)
			try++
			continue
		}

		addr.Proto = protocol
		addr.Port = mapping(protocol, lPort, ePort, desc, lifetime)
		addr.IP, err = client.GetExternalAddress()

		if addr.IsValid() && callback != nil {
			callback(addr)
		}

		select {
		case <-stop:
			goto END
		case <-timer.C:
			timer.Reset(lifetime)
		}
	}

END:
	client.DeletePortMapping(protocol, lPort)
}

// return extPort
func mapping(protocol string, lPort, ePort int, desc string, lifetime time.Duration) int {
	var exp int
	var err error

	client, err := New()
	if err != nil {
		return 0
	}

	if err = client.AddPortMapping(protocol, lPort, ePort, desc, lifetime); err == nil {
		return ePort
	}

	if exp, err = client.AddRandPortMapping(protocol, lPort, desc, lifetime); err == nil {
		return exp
	}

	return 0
}
