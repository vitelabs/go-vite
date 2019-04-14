package api

import (
	"strconv"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vite/net"
)

type NetApi struct {
	net net.Net
	p2p p2p.P2P
	log log15.Logger
}

func NewNetApi(vite *vite.Vite) *NetApi {
	return &NetApi{
		net: vite.Net(),
		p2p: vite.P2P(),
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
	log.Info("SyncInfo")
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

func (n *NetApi) Peers() p2p.NodeInfo {
	return n.p2p.Info()
}
