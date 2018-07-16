package viteconfig

import (
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)

func LoadConfig (fileName string)  {
	config.Load(
		file.NewSource(
			file.WithPath("config/" + fileName + ".json"),
		),
	)
}