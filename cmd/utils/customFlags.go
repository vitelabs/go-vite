package utils

import (
	"strings"
	"os"
	"path"
	"fmt"
	"flag"
	"github.com/vitelabs/go-vite/common"
)

type DirectoryString struct {
	Value string
}

func (self *DirectoryString) String() string {
	return self.Value
}

func (self *DirectoryString) Set(value string) error  {
	self.Value = expandPath(value)
	return nil
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/viteisbest -> /home/username/viteisbest
type DirectoryFlag struct {
	Name string
	Value DirectoryString
	Usage string
}

func (self DirectoryFlag) String() string {
	fmtString := "%s %v\t%v"
	if len(self.Value.Value) > 0 {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(self.Name), self.Value.Value, self.Usage)
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

// called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (self DirectoryFlag) Apply(set *flag.FlagSet) {
	eachName(self.Name, func(name string) {
		set.Var(&self.Value, self.Name, self.Usage)
	})
}

func prefixedNames(fullName string) (prefixed string) {
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	return
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := common.HomeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func (self DirectoryFlag) GetName() string {
	return self.Name
}

func (self *DirectoryFlag) Set(value string) {
	self.Value.Value = value
}
