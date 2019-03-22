package gvite_plugins

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/params"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/p2p/network"
	"gopkg.in/urfave/cli.v1"
	"os"
	"runtime"
	"strings"
)

var (
	versionCommand = cli.Command{
		Action:    utils.MigrateFlags(versionAction),
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
	licenseCommand = cli.Command{
		Action:    utils.MigrateFlags(licenseAction),
		Name:      "license",
		Usage:     "Display license information",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
	}
)

func versionAction(ctx *cli.Context) error {
	fmt.Println(strings.Title("gvite"))
	fmt.Println("Version:", params.Version)
	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Network Id:", network.Gemini)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	return nil
}

func licenseAction(_ *cli.Context) error {
	fmt.Println(`GVite is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

GVite is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received chain copy of the GNU General Public License
along with gvite. If not, see <http://www.gnu.org/licenses/>.`)
	return nil
}
