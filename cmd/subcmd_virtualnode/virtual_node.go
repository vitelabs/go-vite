package subcmd_virtualnode

import (
	"fmt"
	"math/big"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/config"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces/core"
)

var (
	VirtualNodeCommand = cli.Command{
		Action:      utils.MigrateFlags(startVirtualNode),
		Name:        "virtual",
		Usage:       "start virtual node",
		Flags:       append(utils.ConfigFlags, richFlag),
		Category:    "LOCAL COMMANDS",
		Description: `Load ledger.`,
	}
)
var (
	richFlag = cli.StringFlag{
		Name:  "rich",
		Usage: "rich address",
	}
)

func startVirtualNode(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	nodeManager, err := nodemanager.NewDefaultNodeManager(ctx, nodemanager.FullNodeMaker{})
	if err != nil {
		return fmt.Errorf("new node error, %+v", err)
	}
	if ctx.GlobalIsSet(richFlag.GetName()) {
		rich := ctx.GlobalString(richFlag.GetName())
		richAddresses(nodeManager.Node().ViteConfig(), []types.Address{types.HexToAddressPanic(rich)})
	}
	return nodeManager.Start()
}

func richAddresses(cfg *config.Config, rich []types.Address) {
	tokenId := core.ViteTokenId
	stakeAmount := big.NewInt(0).Mul(big.NewInt(100000), new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil)) // 10w * 10**18
	for _, addr := range rich {
		cfg.Genesis.QuotaInfo.StakeBeneficialMap[addr.String()] = stakeAmount
		_, ok := cfg.Genesis.AccountBalanceMap[addr.String()]
		if !ok {
			cfg.Genesis.AccountBalanceMap[addr.String()] = make(map[string]*big.Int)
		}
		cfg.Genesis.AccountBalanceMap[addr.String()][tokenId.String()] = stakeAmount
	}
}
