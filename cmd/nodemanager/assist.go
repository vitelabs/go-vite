package nodemanager

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/node"
)

var (
	log = log15.New("module", "gvite/node_manager")
)

// start node
func StartNode(node *node.Node) error {
	// Prepare the node
	log.Info(fmt.Sprintf("Starting prepare node..."))
	if err := node.Prepare(); err != nil {
		log.Error(fmt.Sprintf("Failed to prepare node, %v", err))
		fmt.Println(fmt.Sprintf("Failed to prepare node, %v", err))
		return err
	}

	// Start the node
	log.Info(fmt.Sprintf("Starting Node..."))
	if err := node.Start(); err != nil {
		fmt.Println(fmt.Sprintf("Failed to start node, %v", err))
		common.Crit(fmt.Sprintf("Failed to start node, %v", err))
	}

	node.Wait()
	return nil
}

// wait the node to stop
func WaitNode(node *node.Node) {
	node.Wait()
}

// stop the node
func StopNode(node *node.Node) {
	log.Warn("Stopping node...")

	//Stop the node Extenders
	log.Warn("Stopping node extenders...")

	if err := node.Stop(); err != nil {
		log.Error(fmt.Sprintf("Failed to stop node, %v", err))
	}
}
