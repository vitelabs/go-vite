package main

import "gopkg.in/urfave/cli.v1"

var (
	walletCmd = cli.Command{
		Name:  "Wallet",
		Usage: "Manage Vite wallets",
	}
	walletBatchCmd = cli.Command{
		Name:  "WalletBatch",
		Usage: "Manage Vite wallets",
	}
)
