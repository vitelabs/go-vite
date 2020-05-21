package main

import (
	_ "net/http/pprof"

	"github.com/vitelabs/go-vite/vite/version"

	"github.com/vitelabs/go-vite/cmd/gvite_plugins"
)

// gvite is the official command-line client for Vite

func main() {
	version.PrintBuildVersion()
	gvite_plugins.Loading()
}
