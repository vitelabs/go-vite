package gvite_plugins

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/node"
	"github.com/vitelabs/go-vite/rpc"
)

var (
	jsFlags = utils.MergeFlags(configFlags, generalFlags, p2pFlags, ipcFlags, httpFlags, wsFlags, consoleFlags, producerFlags, logFlags, vmFlags, netFlags, statFlags)
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

// remoteConsole will connect to a remote gvite instance, attaching a JavaScript console to it.
func remoteConsoleAction(ctx *cli.Context) error {

	// Attach to a remotely running gvite instance and start the JavaScript console
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
		log.Error(fmt.Sprintf("Unable to attach to remote gvite: %v", err))
		return err
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
