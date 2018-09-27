package pool

import "github.com/vitelabs/go-vite/vite/net"

type tools struct {
	// if address == nil, snapshot tools
	// else account fetcher
	fetcher commonSyncer
	rw      chainRw
}

func newTools(f commonSyncer, rw chainRw) *tools {
	self := &tools{}
	self.fetcher = f
	self.rw = rw
	return self
}

type syncer interface {
	net.Broadcaster
	net.Fetcher
	net.Subscriber
}
