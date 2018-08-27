package api

type P2PApi interface {
	// reply true or false
	NetworkAvailable(noop interface{}, reply *string) error
	// reply an int value represents PeersCount
	PeersCount(noop interface{}, reply *string) error
}
