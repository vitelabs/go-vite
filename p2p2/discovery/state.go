package discovery

// state is mean the node state, is follow the `Finite State Machine` mode.
// node can stay one of the following 3 states:

// fresh ------> checked <------> dirty

// 1. fresh, in this state nodes will be put in a stage to check

// 2. checked, in this state nodes will be put into the router table

// 3. dirty, get different NodeID or different Address of this node, check the old Node.
// if check success, then discard the new ID/Address. whereas use the following 2 strategy:
// a. if new ID/Address is get from a ping message, update Node to new ID/Address.
// b. if new ID/Address is get from a neighbors message, then check the new ID/Address:
//  b1. check success, update Node to new ID/Address
//  b2. check failed, discard the new ID/Address, keep the old Node
type state interface {
	handle(cd code) (state, error)
	id() nodeState
}

type nodeState byte

const (
	nodeStateFresh nodeState = 1 << iota
	nodeStateChecked
	nodeStateDirty
)

func (n nodeState) id() nodeState {
	return n
}

type fresh struct {
	nodeState
}

type checked struct {
	nodeState
}

type dirty struct {
	nodeState
}
