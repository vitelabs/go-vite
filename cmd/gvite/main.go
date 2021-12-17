package main

import (
	_ "net/http/pprof"

	"github.com/vitelabs/go-vite/v2/cmd/gvite_plugins"
	"github.com/vitelabs/go-vite/v2/version"
)

// gvite is the official command-line client for Vite

func main() {
	version.PrintBuildVersion()
	gvite_plugins.Loading()
}
