package nodemanager

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
)

var (
	extenderLog     = log15.New("module", "gvite/node_extender")
	NodeExtenderMap = make(map[string]NodeExtender)
)

type NodeExtender interface {
	Start(node *node.Node) error
	Stop() error
}

func Register(extenderName string, extender NodeExtender) {
	NodeExtenderMap[extenderName] = extender
}

func startNodeExtenders(node *node.Node) {
	for name, extender := range NodeExtenderMap {
		if err := extender.Start(node); err != nil {
			extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Star error %v", name, err))
			if err := extender.Stop(); err != nil {
				extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Stop error %v", name, err))
			}
		}
	}
}

func stopNodeExtenders(node *node.Node) {
	for name, extender := range NodeExtenderMap {
		if err := extender.Stop(); err != nil {
			extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Stop error %v", name, err))
		}
	}
}
