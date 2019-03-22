package main

import (
	"net"
	"sync"
)

type IPFlow struct {
	maxConnPerIP int
	ips          map[string]int // ip: conn count
	mu           sync.Mutex
}

func newIPFlow(max int) *IPFlow {
	return &IPFlow{
		maxConnPerIP: max,
		ips:          make(map[string]int),
	}
}

func (i *IPFlow) enter(ip net.IP) bool {
	str := ip.String()

	i.mu.Lock()
	defer i.mu.Unlock()

	if n, ok := i.ips[str]; ok {
		if n < i.maxConnPerIP {
			i.ips[str] = n + 1
			return true
		}

		return false
	} else {
		i.ips[str] = 1
		return true
	}
}

func (i *IPFlow) leave(ip net.IP) {
	str := ip.String()

	i.mu.Lock()
	defer i.mu.Unlock()

	if n, ok := i.ips[str]; ok {
		i.ips[str] = n - 1
	}
}
