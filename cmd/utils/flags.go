package utils

import (
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"os"
	"github.com/vitelabs/go-vite/cmd/params"
	"github.com/vitelabs/go-vite/common"
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

var (
	// General settings
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "use for store all files",
		Value: DirectoryString{common.DefaultDataDir()},
	}
	KeystoreDirFlag = DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	PasswordFileFlag = cli.StringFlag{
		Name: "password",
		Usage: "password file to use for non-interactive password input",
		Value: "",
	}
)
