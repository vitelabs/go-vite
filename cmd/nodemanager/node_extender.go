package nodemanager

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
)

var (
	extenderLog     = log15.New("module", "gvite/node_extender")
	nodeExtenderMap = make(map[string]NodeExtender)
)

type NodeExtender interface {
	Prepare(node *node.Node) error
	Start(node *node.Node) error
	Stop(node *node.Node) error
}

func Register(extenderName string, extender NodeExtender) {
	nodeExtenderMap[extenderName] = extender
}

func prepareNodeExtenders(node *node.Node) {
	for name, extender := range nodeExtenderMap {
		if err := extender.Prepare(node); err != nil {
			extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Prepare error %v", name, err))
			if err := extender.Stop(node); err != nil {
				extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Stop error %v", name, err))
			}
		}
	}
}

func startNodeExtenders(node *node.Node) {
	for name, extender := range nodeExtenderMap {
		if err := extender.Start(node); err != nil {
			extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Start error %v", name, err))
			if err := extender.Stop(node); err != nil {
				extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Stop error %v", name, err))
			}
		}
	}
}

func stopNodeExtenders(node *node.Node) {
	for name, extender := range nodeExtenderMap {
		if err := extender.Stop(node); err != nil {
			extenderLog.Error(fmt.Sprintf("ExtenderName:%v ,Stop error %v", name, err))
		}
	}
}
