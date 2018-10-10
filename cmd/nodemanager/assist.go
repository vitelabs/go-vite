package nodemanager

import (
	"fmt"
	"github.com/vitelabs/go-vite/node"
	"os"
	"os/signal"
	"syscall"
)

// start node
func StartNode(node *node.Node) {

	// Start the node
	log.Info(fmt.Sprintf("Begin StartNode... "))
	if err := node.Start(); err != nil {
		log.Error("Error staring protocol node: %v", err)
	}

	// Listening event closes the node
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)
		<-c
		go func() {
			log.Info(fmt.Sprintf("Begin StopNode..."))
			node.Stop()
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

	node.Stop()
}
