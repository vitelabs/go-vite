package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpcapi/impl"
	"github.com/vitelabs/go-vite/vite"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
)

var (
	accountCommand = cli.Command{
		Name:  "account",
		Usage: "Manage accounts",
		//Category: "ACCOUNT COMMANDS",
		Description: `Manage accounts, include list all account, create a new account,
		and import a private key into a new account.`,
		Subcommands: []cli.Command{
			{
				Name:   "list",
				Usage:  "List all existing accounts",
				Action: accountList,
				//Flags: []cli.Flag {
				//	utils.DataDirFlag,
				//	utils.WalletDirFlag,
				//},
				Description: "Print all accounts",
			},
			{
				Name:   "new",
				Usage:  "Create a new account",
				Action: accountNew,
				//Flags: []cli.Flag {
				//	utils.DataDirFlag,
				//	utils.WalletDirFlag,
				//},
				Description: `gvite account new
create a new account with password and print account address`,
			},
			{
				Name:   "import",
				Usage:  "import a private key into a new account",
				Action: accountImport,
				//Flags: []cli.Flag {
				//	utils.DataDirFlag,
				//	utils.WalletDirFlag,
				//	utils.PasswordFileFlag,
				//},
				ArgsUsage: "<keyfile>",
				Description: `gvite account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.`,
			},
		},
	}
)

func accountList(ctx *cli.Context) error {
	mainLog := log15.New("module", "accountcmd/accountList")

	localconfig := makeConfigNode()
	//localconfig := config.GlobalConfig
	vnode, err := vite.New(localconfig)
	if err != nil {
		mainLog.Error(err.Error())
	}

	walletApi := impl.NewWalletApi(vnode)

	var result = new(string)
	walletApi.ListAddress(nil, result)
	fmt.Println(*result)

	//index := 0
	//for _, addr := range vnode.WalletManager().KeystoreManager.Addresses() {
	//	fmt.Printf("Account #%d: {%s}\n", index, addr)
	//	index ++
	//}
	return nil
}

// getPassPhrase retrieves the password associated with an account.
func getPassPhrase(prompt string, confirmation bool) string {
	log := log15.New("module", "accountcmd/accountNew")

	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		log.Error("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			log.Error("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			log.Error("Passphrases do not match")
		}
	}
	return password
}

func accountNew(ctx *cli.Context) error {
	mainLog := log15.New("module", "accountcmd/accountNew")

	localconfig := makeConfigNode()
	vnode, err := vite.New(localconfig)
	if err != nil {
		mainLog.Error(err.Error())
	}

	walletApi := impl.NewWalletApi(vnode)

	var result = new(string)
	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true)
	passphare := []string{password}
	walletApi.NewAddress(passphare, result)
	fmt.Println(*result)
	return nil
}

func getPassFromFile(fileName string) (string, error) {
	buf, _ := ioutil.ReadFile(fileName)
	return string(buf), nil
}

func accountImport(ctx *cli.Context) error {
	mainLog := log15.New("module", "accountcmd/accountImport")

	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		mainLog.Error("keyfile must be given as argument")
	}

	localconfig := makeConfigNode()
	vnode, err := vite.New(localconfig)
	if err != nil {
		mainLog.Error(err.Error())
	}

	walletApi := impl.NewWalletApi(vnode)

	var result = new(string)
	password, err := getPassFromFile(keyfile)
	if err != nil {
		mainLog.Error("read password from file error.")
	}

	passphare := []string{password}
	walletApi.NewAddress(passphare, result)
	fmt.Println(*result)
	return nil
}
