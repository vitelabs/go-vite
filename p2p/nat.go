package p2p

import (
	"github.com/jackpal/gateway"
	"github.com/jackpal/go-nat-pmp"
	"fmt"
	"time"
	"net"
	"sync"
)

type natClient struct {
	mutex sync.Mutex
	client *natpmp.Client
}

func (c *natClient) getClient() (*natpmp.Client, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client == nil {
		gateway.DiscoverGateway()
		gatewayIP, err := gateway.DiscoverGateway()
		if err != nil {
			return nil, err
		}

		c.client = natpmp.NewClient(gatewayIP)
	}

	return c.client, nil
}

var gclt natClient

var defaultLifetime = 15 * time.Minute

func getExtIP() (net.IP, error) {
	client, err := gclt.getClient()

	if err != nil {
		return nil, err
	}

	response, err := client.GetExternalAddress()
	if err != nil {
		return nil, err
	}

	ext := response.ExternalIPAddress

	ip := net.ParseIP(string(ext[:]))

	if ip == nil {
		return nil, fmt.Errorf("get invalid external ip: %v\n", ext)
	}

	return ip, nil
}

func natMap(stop <- chan struct{}, protocol string, localIp, extIP int, lifetime time.Duration, errch chan<- error) {
	client, err := gclt.getClient()

	hasErrch := errch != nil
	if hasErrch {
		defer close(errch)
	}

	if err != nil && hasErrch {
		errch <- err
		return
	}

	if lifetime == 0 {
		lifetime = defaultLifetime
	}

	if _, err = client.AddPortMapping(protocol, localIp, extIP, int(lifetime.Seconds())); err != nil {
		fmt.Printf("nat map error %v\n", err)
		if hasErrch {
			errch <- err
		}
	}

	timer := time.NewTimer(lifetime)
	defer timer.Stop()

	for {
		select {
		case <- stop:
			return
		case <- timer.C:
			if _, err = client.AddPortMapping(protocol, localIp, extIP, int(lifetime.Seconds())); err != nil {
				fmt.Printf("nat map error %v\n", err)
				if hasErrch {
					errch <- err
				}
			}
			timer.Reset(lifetime)
		}
	}
}
