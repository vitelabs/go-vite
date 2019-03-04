package config

import (
	"path/filepath"

	"github.com/vitelabs/go-vite/config/biz"
)

type Config struct {
	*Producer   `json:"Producer"`
	*Chain      `json:"Chain"`
	*Vm         `json:"Vm"`
	*Subscribe  `json:"Subscribe"`
	*Net        `json:"Net"`
	*biz.Reward `json:"Reward"`
	*Genesis    `json:"Genesis"`

	// global keys
	DataDir string `json:"DataDir"`
	//Log level
	LogLevel string `json:"LogLevel"`
}

func (c Config) RunLogDir() string {
	return filepath.Join(c.DataDir, "runlog")
}
