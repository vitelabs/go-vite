package utils

import (
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"os"
	"os/signal"
	"syscall"
)

func StartNode(node *node.Node) {

	if err := node.Start(); err != nil {
		log15.Error("Error staring protocol node: %v", err)
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)
		<-c
		log15.Info("Got interrupt, shutting down...")
		go node.Stop()
		for i := 10; i > 0; i-- {
			<-c
			if i > 1 {
				log15.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}

	}()
}
