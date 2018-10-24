package gvite_plugins

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/node"
	"github.com/vitelabs/go-vite/rpc"
	"gopkg.in/urfave/cli.v1"
)

var (
	jsFlags = utils.MergeFlags(configFlags, generalFlags, p2pFlags, ipcFlags, httpFlags, wsFlags, consoleFlags, producerFlags, logFlags, vmFlags, netFlags, statFlags)

	//local
	consoleCommand = cli.Command{
		Action:   utils.MigrateFlags(localConsoleAction),
		Name:     "console",
		Usage:    "Start an interactive JavaScript environment",
		Flags:    jsFlags,
		Category: "CONSOLE COMMANDS",
		Description: `
The GVite console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/vitelabs/go-vite/wiki/JavaScript-Console.`,
	}

	//local
	javascriptCommand = cli.Command{
		Action:    utils.MigrateFlags(ephemeralConsoleAction),
		Name:      "js",
		Usage:     "Execute the specified JavaScript files",
		ArgsUsage: "<jsfile> [jsfile...]",
		Flags:     jsFlags,
		Category:  "CONSOLE COMMANDS",
		Description: `
The JavaScript VM exposes a node admin interface as well as the Ðapp
JavaScript API. See https://github.com/vitelabs/go-vite/wiki/JavaScript-Console`,
	}

	//remote
	attachCommand = cli.Command{
		Action:    utils.MigrateFlags(remoteConsoleAction),
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		ArgsUsage: "[endpoint]",
		Flags:     append(consoleFlags, utils.DataDirFlag),
		Category:  "CONSOLE COMMANDS",
		Description: `
The GVite console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/vitelabs/go-vite/wiki/JavaScript-Console.
This command allows to open a console on a running gvite node.`,
	}
)

// localConsole starts a new gvite node, attaching a JavaScript console to it at the same time.
func localConsoleAction(ctx *cli.Context) error {

	// Create and start the node based on the CLI flags
	nodeManager := nodemanager.NewConsoleNodeManager(ctx, nodemanager.FullNodeMaker{})
	nodeManager.Start()
	defer nodeManager.Stop()

	// Attach to the newly started node and start the JavaScript console
	client, err := nodeManager.Node().Attach()
	if client == nil || err != nil {
		log.Error(fmt.Sprintf("Failed to attach to the inproc geth: %v", err))
	}

	config := console.Config{
		DataDir: nodeManager.Node().Config().DataDir,
		DocRoot: ctx.GlobalString(utils.JSPathFlag.Name),
		Client:  client,
		Preload: makeConsolePreLoads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to start the JavaScript console: %v", err))
	}
	defer console.Stop(false)

	// If only a short execution was requested, evaluate and return
	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}

// ephemeralConsole starts a new geth node, attaches an ephemeral JavaScript
// console to it, executes each of the files specified as arguments and tears
// everything down.
func ephemeralConsoleAction(ctx *cli.Context) error {

	// Create and start the node based on the CLI flags
	nodeManager := nodemanager.NewConsoleNodeManager(ctx, nodemanager.FullNodeMaker{})
	nodeManager.Start()
	defer nodeManager.Stop()

	// Attach to the newly started node and start the JavaScript console
	client, err := nodeManager.Node().Attach()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to attach to the inproc geth: %v", err))
	}

	config := console.Config{
		DataDir: nodeManager.Node().Config().DataDir,
		DocRoot: ctx.GlobalString(utils.JSPathFlag.Name),
		Client:  client,
		Preload: makeConsolePreLoads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to start the JavaScript console: %v", err))
	}
	defer console.Stop(false)

	// Evaluate each of the specified JavaScript files
	for _, file := range ctx.Args() {
		if err = console.Execute(file); err != nil {
			log.Error(fmt.Sprintf("Failed to execute %s: %v", file, err))
		}
	}
	// Wait for pending callbacks, but stop for Ctrl-C.
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-abort
		os.Exit(0)
	}()
	console.Stop(true)

	return nil
}

// remoteConsole will connect to a remote geth instance, attaching a JavaScript console to it.
func remoteConsoleAction(ctx *cli.Context) error {

	// Attach to a remotely running geth instance and start the JavaScript console
	//gvite attach ipc:/some/custom/path
	//gvite attach http://191.168.1.1:8545
	//gvite attach ws://191.168.1.1:8546
	dataDir := makeDataDir(ctx)
	attachEndpoint := ctx.Args().First()
	if attachEndpoint == "" {
		attachEndpoint = defaultAttachEndpoint(dataDir)
	}
	client, err := dialRPC(dataDir, attachEndpoint)
	if err != nil {
		log.Error(fmt.Sprintf("Unable to attach to remote geth: %v", err))
	}
	config := console.Config{
		DataDir: dataDir,
		DocRoot: ctx.GlobalString(utils.JSPathFlag.Name),
		Client:  client,
		Preload: makeConsolePreLoads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to start the JavaScript console: %v", err))
	}
	defer console.Stop(false)

	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}

// dialRPC returns a RPC client which connects to the given endpoint.
// The check for empty endpoint implements the defaulting logic for "gvite attach" with no argument.
func dialRPC(dataDir string, endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		config := &node.Config{
			DataDir: dataDir,
			IPCPath: endpoint,
		}
		endpoint = config.IPCEndpoint()
	}
	return rpc.Dial(endpoint)
}

func defaultAttachEndpoint(dataDir string) string {
	return fmt.Sprintf("%s/gvite.ipc", dataDir)
}

func makeDataDir(ctx *cli.Context) string {
	path := node.DefaultDataDir()
	netId := ctx.GlobalUint(utils.NetworkIdFlag.Name)
	if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		path = ctx.GlobalString(utils.DataDirFlag.Name)
	}
	if path != "" {
		if ctx.GlobalBool(utils.MainNetFlag.Name) || netId == 1 {
			path = filepath.Join(path, "maindata")
		} else if ctx.GlobalBool(utils.TestNetFlag.Name) || netId == 2 {
			path = filepath.Join(path, "testdata")
		} else if ctx.GlobalBool(utils.DevNetFlag.Name) || netId > 2 || netId < 1 {
			path = filepath.Join(path, "devdata")
		} else {
			path = filepath.Join(path, "devdata")
		}
	}
	return path
}

// MakeConsolePreLoads retrieves the absolute paths for the console JavaScript scripts to preload before starting.
func makeConsolePreLoads(ctx *cli.Context) []string {
	// Skip preLoading if there's nothing to preload
	if ctx.GlobalString(utils.PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preLoads := []string{}

	jsPath := ctx.GlobalString(utils.JSPathFlag.Name)
	for _, file := range strings.Split(ctx.GlobalString(utils.PreloadJSFlag.Name), ",") {
		preLoads = append(preLoads, utils.AbsolutePath(jsPath, strings.TrimSpace(file)))
	}
	return preLoads
}
