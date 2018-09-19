package utils

import "github.com/vitelabs/go-vite/node"

func StartNode(node *node.Node) {

	if err := node.Start(); err != nil {
		node.Logger.Error("Error staring protocol node: %v", err)
	}

	//TODO add node.Stop()

}
