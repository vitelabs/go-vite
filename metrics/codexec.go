package metrics

import (
	"strings"
	"time"
)

var (
	CodexecRegistry       = NewPrefixedChildRegistry(DefaultRegistry, "/codexec")
	TimeconsumingRegistry = NewPrefixedChildRegistry(CodexecRegistry, "/timeconsuming")
	BranchRegistry        = NewPrefixedChildRegistry(CodexecRegistry, "/branch")
)

func TimeConsuming(names []string, sinceTime time.Time) {
	var name string
	for _, v := range names {
		name += "/" + strings.ToLower(v)
	}
	if timer, ok := GetOrRegisterResettingTimer(name, TimeconsumingRegistry).(*StandardResettingTimer); timer != nil && ok {
		timer.UpdateSince(sinceTime)
	} else {
		log.Error("moduleRegistry is nil")
	}
}
