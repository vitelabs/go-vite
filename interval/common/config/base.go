package config

import (
	"errors"
	"os/user"
)

type Base struct {
	DataDir string `yaml:"dataDir"`
	Log     string `yaml:"log"`
}

func (cfg *Base) Check() error {
	if len(cfg.DataDir) == 0 {
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
