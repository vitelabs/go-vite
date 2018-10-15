package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/node"
	"testing"
)

func TestConsole(t *testing.T) {

	// Attach to a remotely running geth instance and start the JavaScript console
	//gvite attach ipc:/some/custom/path
	//gvite attach http://191.168.1.1:8545
	//gvite attach ws://191.168.1.1:8546

	config := console.Config{
		DataDir: node.DefaultDataDir(),
		DocRoot: "",
		Client:  nil,
		Preload: nil,
	}

	console, err := console.New(config)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to start the JavaScript console: %v", err))
	}
	defer console.Stop(false)

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

}
