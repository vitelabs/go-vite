package vm

import (
	"github.com/vitelabs/go-vite/metrics"
)

var vmImpossible = metrics.GetOrRegisterMeter("/possible/vm", metrics.BranchRegistry)
