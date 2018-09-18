package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/params"
	"gopkg.in/urfave/cli.v1"
	"runtime"
	"strings"
)

var (
	versionCommand = cli.Command{
		Action:    version,
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		//Category: "AIDCMD COMMANDS",
	}
)

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(ClientIdentifier))
	fmt.Println("version:", params.Version)
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Println("GOROOT:", runtime.GOROOT())

	return nil
}
