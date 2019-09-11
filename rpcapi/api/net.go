package api

import (
	"strconv"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/net/vnode"
	"github.com/vitelabs/go-vite/vite"
)

type NetApi struct {
	net net.Net
	log log15.Logger
}

func NewNetApi(vite *vite.Vite) *NetApi {
	return &NetApi{
		net: vite.Net(),
		log: log15.New("module", "rpc_api/net_api"),
	}
}

type SyncInfo struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Current string `json:"current"`
	State   uint   `json:"state"`
	Status  string `json:"status"`
}

func (n *NetApi) SyncInfo() SyncInfo {
	s := n.net.Status()

	return SyncInfo{
		From:    strconv.FormatUint(s.From, 10),
		To:      strconv.FormatUint(s.To, 10),
		Current: strconv.FormatUint(s.Current, 10),
		State:   uint(s.State),
		Status:  s.State.String(),
	}
}

func (n *NetApi) SyncDetail() net.SyncDetail {
	return n.net.Detail()
}

// Peers is for old api
func (n *NetApi) Peers() net.NodeInfo {
	return n.net.Info()
}

func (n *NetApi) PeerCount() int {
	return n.net.PeerCount()
}

func (n *NetApi) NodeInfo() net.NodeInfo {
	return n.net.Info()
}

type Nodes struct {
	Count int
	Nodes []*vnode.Node
}

func (n *NetApi) Nodes() Nodes {
	nodes := n.net.Nodes()
	return Nodes{
		Nodes: nodes,
		Count: len(nodes),
	}
}
