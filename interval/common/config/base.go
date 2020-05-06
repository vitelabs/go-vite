package config

import (
	"errors"
	"os/user"
)

type Base struct {
	DataDir string `yaml:"dataDir"`
	Log     string `yaml:"log"`
}

func (self *Base) Check() error {
	if len(self.DataDir) == 0 {
		return errors.New("base[DataDir] must be set")
	}

	return nil
}

func init() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	HomeDir = usr.HomeDir
}

var HomeDir = ""
