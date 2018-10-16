package nodemanager

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"os"
	"os/signal"
	"syscall"
)

var (
	log = log15.New("module", "gvite/node_manager")
)

// start node
func StartNode(node *node.Node) {

	// Start the node
	log.Info(fmt.Sprintf("Begin StartNode... "))
	if err := node.Start(); err != nil {
		log.Error(fmt.Sprintf("Failed to start node， %v", err))
		fmt.Println(fmt.Sprintf("Failed to start node， %v", err))
	} else {
		fmt.Println("Start the Node success!!!")
	}

	// Listening event closes the node
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		defer signal.Stop(c)
		<-c
		fmt.Println("Prepare Stop the Node...")

		go func() {
			StopNode(node)
		}()

		for i := 10; i > 0; i-- {
			<-c
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
	}()

}

// wait the node to stop
func WaitNode(node *node.Node) {

	node.Wait()
}

// stop the node
func StopNode(node *node.Node) {
	fmt.Sprintf("Stop the Node...")
	log.Warn("Stop the Node...")
	if err := node.Stop(); err != nil {
		log.Error(fmt.Sprintf("Node stop error: %v", err))
	}
}
