package params

import "github.com/vitelabs/go-vite"

var Version = func() string {
	return govite.VITE_BUILD_VERSION
}()
