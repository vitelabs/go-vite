package api

import (
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vite/net"
	"strconv"
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
	StartHeight      string `json:"startHeight"`
	TargetHeight     string `json:"targetHeight"`
	CurrentHeight    string `json:"currentHeight"`
	IsFirstSyncDone  bool   `json:"isFirstSyncDone"`
	IsStartFirstSync bool   `json:"isStartFirstSync"`
}

func (n *NetApi) SyncInfo() *SyncInfo {
	s := n.net.Status()

	return &SyncInfo{
		StartHeight:      strconv.FormatUint(s.From, 10),
		TargetHeight:     strconv.FormatUint(s.To, 10),
		CurrentHeight:    strconv.FormatUint(s.Current, 10),
		IsFirstSyncDone:  s.State == net.Syncdone,
		IsStartFirstSync: true,
	}
}
