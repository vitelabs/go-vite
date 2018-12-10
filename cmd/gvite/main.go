package main

import (
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/cmd/gvite_plugins"
	_ "net/http/pprof"
)

// gvite is the official command-line client for Vite

func main() {
	govite.PrintBuildVersion()
	gvite_plugins.Loading()

}
