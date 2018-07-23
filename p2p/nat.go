package p2p

import (
	"fmt"
	"time"
	"net"
	"sync"
	"github.com/syaka-yin/go-nat"
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
		return
	}

	if lifetime == 0 {
		lifetime = defaultLifetime
	}

	mp := func() {
		if err = client.AddPortMapping(protocol, lPort, ePort, "vite", lifetime); err != nil {
			fmt.Printf("nat map error %v\n", err)
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
