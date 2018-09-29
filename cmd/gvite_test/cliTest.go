package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/params"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/log15"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"sort"
)

var (
	app = NewApp()

	// Network Settings
	IdentityFlag = cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}

	nodeFlags = []cli.Flag{
		utils.IdentityFlag,
	}

	log = log15.New("module", "main")
)

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	app.Email = ""
	app.Version = params.Version
	app.Usage = "the go-vite cli application"
	return app
}

func init() {
	//Initialize the CLI app and start Gvite
	app.Action = gvite
	//app.HideVersion = true
	app.Copyright = "Copyright 2018-2024 The go-vite Authors"

	testCommand := cli.Command{
		Action:    testAction,
		Name:      "test",
		Usage:     "Display test information",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
	}

	app.Commands = []cli.Command{
		testCommand,
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)

	//TODO missing app.Before
	//TODO missing app.After
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Warn(fmt.Sprintln("app.Run.error", err))
		os.Exit(1)
	}
}

func gvite(ctx *cli.Context) error {

	//TODO invalid is why
	log.Info("gvite here")
	log.Info(fmt.Sprintf("os.args.len is %v", os.Args))
	log.Info(fmt.Sprintf("ctx.args.len is %v", ctx.Args()))
	log.Info(fmt.Sprintf("IdentityFlag.Name is %v", ctx.GlobalString(IdentityFlag.Name)))

	if args := ctx.Args(); len(args) > 0 {
		log.Info(fmt.Sprintf("invalid command: %q", args[0]))

		return fmt.Errorf("invalid command: %q", args[0])
	}

	doSomething(ctx)

	return nil
}

func doSomething(ctx *cli.Context) {
	log.Info("doSomething here")
}

func testAction(ctx *cli.Context) error {
	log.Info("testAction here")
	return nil
}
