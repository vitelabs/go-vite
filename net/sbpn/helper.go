package sbpn

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/discovery"
)

func nodeParser(node *discovery.Node) target {
	var address types.Address
	copy(address[:], node.Ext)

	return target{
		id:      node.ID,
		address: address,
		tcp:     node.TCPAddr(),
	}
}
