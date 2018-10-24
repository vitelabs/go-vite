package api

import (
	"encoding/json"
	"fmt"
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
	From     string `json:"from"`
	To       string `json:"to"`
	Received string `json:"received"`
	Current  string `json:"current"`
	State    uint   `json:"state"`
	Status   string `json:"status"`
}

func (n *NetApi) SyncInfo() *SyncInfo {
	log.Info("SyncInfo")
	s := n.net.Status()

	return &SyncInfo{
		From:     strconv.FormatUint(s.From, 10),
		To:       strconv.FormatUint(s.To, 10),
		Received: strconv.FormatUint(s.Received, 10),
		Current:  strconv.FormatUint(s.Current, 10),
		State:    uint(s.State),
		Status:   s.State.String(),
	}
}

func (n *NetApi) Peers() (ret []string) {
	info := n.net.Info()

	var receiveTotal, handleTotal, discardTotal, sendTotal uint64
	for _, pinfo := range info.Peers {
		receiveTotal += pinfo.Received
		discardTotal += pinfo.Discarded

		for _, num := range pinfo.MsgHandle {
			handleTotal += num
		}

		for _, num := range pinfo.MsgSend {
			sendTotal += num
		}

		if js, err := json.Marshal(pinfo); err == nil {
			ret = append(ret, string(js))
		}
	}

	ret = append(ret, fmt.Sprintf("total received: %d", receiveTotal))
	ret = append(ret, fmt.Sprintf("total handled: %d", handleTotal))
	ret = append(ret, fmt.Sprintf("total discarded: %d", discardTotal))
	ret = append(ret, fmt.Sprintf("total send: %d", sendTotal))

	return
}

func (n *NetApi) PeersCount() uint {
	info := n.net.Info()
	return uint(len(info.Peers))
}
