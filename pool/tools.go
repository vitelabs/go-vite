package pool

import "github.com/vitelabs/go-vite/vite/net"

type tools struct {
	// if address == nil, snapshot tools
	// else account fetcher
	fetcher  commonSyncer
	verifier commonVerifier
	rw       chainRw
}

func newTools(f commonSyncer, v commonVerifier, rw chainRw) *tools {
	self := &tools{}
	self.fetcher = f
	self.verifier = v
	self.rw = rw
	return self
}

type syncer interface {
	net.Broadcaster
	net.Fetcher
	net.Subscriber
}
