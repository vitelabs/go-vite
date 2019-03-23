package discovery

// Finder is the engine to find other nodes in the same network
type Finder interface {
	Start() error
	Stop() error
	// Nodes will supply a batch nodes to caller
	Nodes() []*Node
}

type findNeighors struct {
}

type findWhole struct {
}
