package utils

import (
	"github.com/vitelabs/go-vite/cmd/params"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

var (
	CommandHelpTemplate = `{{.cmd.Name}}{{if .cmd.Subcommands}} command{{end}}{{if .cmd.Flags}} [command options]{{end}} [arguments...]
{{if .cmd.Description}}{{.cmd.Description}}
{{end}}{{if .cmd.Subcommands}}
SUBCOMMANDS:
	{{range .cmd.Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
	{{end}}{{end}}{{if .categorizedFlags}}
{{range $idx, $categorized := .categorizedFlags}}{{$categorized.Name}} OPTIONS:
{{range $categorized.Flags}}{{"\t"}}{{.}}
{{end}}
{{end}}{{end}}`
)

func init() {
	cli.AppHelpTemplate = `{{.Name}} {{if .Flags}}[global options] {{end}}command{{if .Flags}} [command options]{{end}} [arguments...]

VERSION:
   {{.Version}}

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = CommandHelpTemplate
}

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
		Value: DirectoryString{common.DefaultDataDir()}, // TODO Distinguish different environmental addresses
	}

	// Network Settings
	IdentityFlag = cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}
	NetworkIdFlag = cli.UintFlag{
		Name: "networkid", //mapping:p2p.NetID
		Usage: "Network identifier (integer," +
			" 1=MainNet," +
			" 2=Aquarius," +
			" 3=Pisces," +
			" 4=Aries," +
			" 5=Taurus," +
			" 6=Gemini," +
			" 7=Cancer," +
			" 8=Leo," +
			" 9=Virgo," +
			" 10=Libra," +
			" 11=Scorpio," +
			" 12=Sagittarius," +
			" 13=Capricorn,)",
		Value: config.GlobalConfig.NetID,
	}
	MaxPeersFlag = cli.UintFlag{
		Name:  "maxpeers", //mapping:p2p.MaxPeers
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: config.GlobalConfig.MaxPeers,
	}
	MaxPendingPeersFlag = cli.UintFlag{
		Name:  "maxpendpeers", //mapping:p2p.MaxPendingPeers
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: config.GlobalConfig.MaxPeers,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port", //mapping:p2p.Addr
		Usage: "Network listening port",
		Value: 8483,
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekeyhex", //mapping:p2p.PrivateKey
		Usage: "P2P node key as hex",
	}
)

// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(ctx *cli.Context, cfg *config.Config) {

	//Global Config
	if dataDir := ctx.GlobalString(DataDirFlag.Name); len(dataDir) > 0 {
		cfg.DataDir = dataDir
	}

	//Network Config
	if identity := ctx.GlobalString(IdentityFlag.Name); len(identity) > 0 {
		cfg.Name = identity
	}

	if ctx.GlobalIsSet(NetworkIdFlag.Name) {
		cfg.NetID = ctx.GlobalUint(NetworkIdFlag.Name)
	}

	if ctx.GlobalIsSet(MaxPeersFlag.Name) {
		cfg.MaxPeers = ctx.GlobalUint(MaxPeersFlag.Name)
	}

	// TODO p2p will use uint
	if ctx.GlobalIsSet(MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = ctx.GlobalInt(MaxPendingPeersFlag.Name)
	}

	if ctx.GlobalIsSet(ListenPortFlag.Name) {
		cfg.Addr = ctx.GlobalString(ListenPortFlag.Name)
	}

	if ctx.GlobalIsSet(NodeKeyHexFlag.Name) {
		cfg.PrivateKey = ctx.GlobalString(NodeKeyHexFlag.Name)
	}

	//TODO other config missing

}
