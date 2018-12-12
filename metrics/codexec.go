package metrics

import (
	"strings"
	"time"
)

var (
	CodexecRegistry       = NewPrefixedChildRegistry(DefaultRegistry, "/codexec")
	TimeconsumingRegistry = NewPrefixedChildRegistry(CodexecRegistry, "/timeconsuming")
)

func TimeConsuming(moduleName string, funcName string, sinceTime time.Time) {
	mName := "/" + strings.ToLower(moduleName)
	fName := "/" + strings.ToLower(funcName)
	if timer, ok := TimeconsumingRegistry.GetOrRegister(mName+fName, NewTimer()).(*StandardTimer); timer != nil && ok {
		timer.UpdateSince(sinceTime)
	} else {
		log.Error("moduleRegistry is nil")
	}
}
