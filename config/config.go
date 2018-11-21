package config

import (
	"path/filepath"
)

type Config struct {
	*Producer `json:"Producer"`
	*Chain    `json:"Chain"`
	*Vm       `json:"Vm"`
	*Net      `json:"Net"`

	// global keys
	DataDir string `json:"DataDir"`
	//Log level
	LogLevel string `json:"LogLevel"`
}

func (c Config) RunLogDir() string {
	return filepath.Join(c.DataDir, "runlog")
}
