package api

type P2PApi interface {
	// reply true or false
	NetworkAvailable() bool
	// reply an int value represents PeersCount
	PeersCount() int
}
