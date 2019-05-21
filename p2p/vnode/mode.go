package vnode

// NodeMode mean the level of a node in the current hierarchy
// Core nodes works on the highest level, usually are producers
// Relay nodes usually are the standby producers, and partial full nodes (like static nodes)
// Regular nodes usually are the full nodes
// Edge nodes usually are the light nodes
type NodeMode byte

const (
	Edge NodeMode = 1 << iota
	Regular
	Relay
	Core
)

func (n NodeMode) String() string {
	switch n {
	case Edge:
		return "edge"
	case Regular:
		return "regular"
	case Relay:
		return "relay"
	case Core:
		return "core"
	default:
		return "unknown"
	}
}
