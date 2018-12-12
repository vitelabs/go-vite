package vm

import "github.com/vitelabs/go-vite/metrics"

var (
	codexecTimeCounter = metrics.NewRegisteredCounter("/vm/codexec/impossible", nil)
)
