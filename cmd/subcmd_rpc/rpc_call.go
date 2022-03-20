package subcmd_rpc

import (
	"encoding/json"
	"fmt"

	"github.com/vitelabs/go-vite/v2/client"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

var (
	//remote
	RpcCommand = cli.Command{
		Action:      utils.MigrateFlags(remoteCallAction),
		Name:        "rpc",
		Usage:       "Remote Procedure Call",
		ArgsUsage:   "gvite rpc [endpoint] methodName '[args]'",
		Flags:       append(utils.ConsoleFlags, utils.DataDirFlag),
		Category:    "REMOTE COMMANDS",
		Description: `For details, see <https://docs.vite.org/vite-docs/api/rpc>`,
	}
)

// remoteCallAction will connect to a remote gvite instance, and call rpc method.
func remoteCallAction(ctx *cli.Context) error {
	//gvite rpc http://127.0.0.1:48132 ledger_getSnapshotBlockByHeight '[100]'
	endpoint := ctx.Args().First()
	method := ctx.Args().Get(1)
	args_str := ctx.Args().Get(2)

	conn, err := client.NewRpcClient(endpoint)
	if err != nil {
		return err
	}
	var args []interface{}
	if args_str != "" {
		json.Unmarshal([]byte(args_str), &args)
	}

	result, err := conn.RawCall(method, args...)
	if err != nil {
		return err
	}
	result_byt, err := json.Marshal(result)
	if err != nil {
		return err
	}
	// fmt.Println(method, args_str)
	fmt.Println(string(result_byt))
	return nil
}
