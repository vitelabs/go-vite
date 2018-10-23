package main

import (
	"github.com/vitelabs/go-vite/cmd/gvite_plugin"
	_ "net/http/pprof"
)

// gvite is the official command-line client for Vite

func main() {
	gvite_plugin.Loading()
}
