package p2p

import (
	"time"
	"net"
	"sync"
	"github.com/syaka-yin/go-nat"
	"log"
)

type natClient struct {
	mutex sync.Mutex
	nat nat.NAT
}

func (c *natClient) getClient() (nat.NAT, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.nat == nil {
		nat, err := nat.DiscoverGateway()
		if err != nil {
			return nil, err
		}

		c.nat = nat
	}

	return c.nat, nil
}

var gclt natClient

var defaultLifetime = 15 * time.Minute

func getExtIP() (net.IP, error) {
	client, err := gclt.getClient()

	if err != nil {
		return nil, err
	}

	return client.GetExternalAddress()
}

func natMap(stop <- chan struct{}, protocol string, lPort, ePort int, lifetime time.Duration) {
	client, err := gclt.getClient()

	if err != nil {
		log.Printf("nat map error: %v\n", err)
		return
	}

	if lifetime == 0 {
		lifetime = defaultLifetime
	}

	mp := func() {
		if err = client.AddPortMapping(protocol, lPort, ePort, "vite", lifetime); err != nil {
			log.Printf("nat map localPort %d to publicPort %d error: %v\n", lPort, ePort, err)
		} else {
			log.Printf("nat map localPort %d to publicPort %d done %v\n", lPort, ePort, err)
		}
	}
	mp()

	timer := time.NewTimer(lifetime)
	defer timer.Stop()

	for {
		select {
		case <- stop:
			return
		case <- timer.C:
			mp()
			timer.Reset(lifetime)
		}
	}
}
