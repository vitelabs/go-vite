package main

import (
	"github.com/vitelabs/go-vite/cmd/gvite_plugins"
	_ "net/http/pprof"
)

// gvite is the official command-line client for Vite

func main() {
	gvite_plugins.Loading()
}
